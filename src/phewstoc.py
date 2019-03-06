#!/usr/bin/env python3

import argparse
import atexit
import hashlib
import requests
import json
import multiprocessing as mp
import pathlib
import sched
import signal
import time
from collections import deque
from subprocess import PIPE, CalledProcessError, run
from threading import Thread, Lock

import cv2
import face_recognition
import numpy as np
from kodipydent import Kodi

PAUSE_TIMEOUT_SECONDS = 300
OFF_TIMEOUT_SECONDS = 600


class KODI:
    def __init__(self, people, address="localhost"):
        self.people = people
        self.kodi = Kodi(address)

    def login(self, name):
        for person in self.people:
            if person["name"] == name:
                self.kodi.GUI.ShowNotification(title="Welcome", message=name)
                # TODO: Login to KODI with person["name"] and person["password"] here.
                break

    def pause(self, name):
        self.kodi.GUI.ShowNotification(title="Are you there?", message="Show your face %s" % name)
        players = self.kodi.Player.GetActivePlayers()
        for player in players["result"]:
            properties = self.kodi.Player.GetProperties(playerid=player["playerid"], properties=["speed"])
            if properties["result"]["speed"] > 0:
                self.kodi.Player.PlayPause(player["playerid"])

    def play(self, name):
        self.kodi.GUI.ShowNotification(title="Ah, there you are!", message="Nice face, you look beautiful %s" % name)
        players = self.kodi.Player.GetActivePlayers()
        for player in players["result"]:
            properties = self.kodi.Player.GetProperties(playerid=player["playerid"], properties=["speed"])
            if properties["result"]["speed"] <= 0:
                self.kodi.Player.PlayPause(player["playerid"])


class TV:
    def __init__(self,
                 device,
                 kodi,
                 pause_timeout=PAUSE_TIMEOUT_SECONDS,
                 off_timeout=OFF_TIMEOUT_SECONDS,
                 aux_on_command="",
                 aux_off_command=""):
        self.kodi = kodi
        self.device = device
        self.pause_timeout = pause_timeout
        self.off_timeout = off_timeout
        self.is_on = False
        self.resume = False
        self.lock = Lock()

        self.aux_off_command = aux_off_command
        self.aux_on_command = aux_on_command

        self.scheduler = sched.scheduler(time.time, time.sleep)

        thread = Thread(target=self._run_scheduler)
        thread.start()

    @staticmethod
    def _run_aux_command(command):
        proc = run(command.split(" "), check=True, stdout=PIPE)
        if len(proc.stdout) > 0:
            print("AUX command '%s' returned output: %s" % (command, proc.stdout.decode('utf-8')))

    def _cec_send(self, command):
        run(["cec-client", "-s", "-d", str(self.device)],
            stdout=PIPE,
            input=command,
            universal_newlines=True,
            check=True)

    def _run_scheduler(self):
        while True:
            self.scheduler.run()

    def _cec_off(self):
        self._cec_send("standby 0")

    def _cec_on(self):
        self._cec_send("on 0")

    def _off(self):
        print("Turning off TV")
        try:
            print("Sending CEC 'standby' signal")
            self._cec_off()
            self.is_on = False
            self.resume = False
        except CalledProcessError as e:
            print("Failed to turn off TV with CEC: %s" % e)

        if self.aux_off_command != "":
            try:
                print("Running auxiliary 'off' command")
                self._run_aux_command(self.aux_off_command)
            except CalledProcessError as e:
                print("Failed to run auxiliary 'off' command: %s" % e)

    def _pause(self, name):
        print("Pausing TV playback")
        self.resume = True
        self.kodi.pause(name)

    def _play(self, name):
        if self.resume:
            print("Resuming TV playback")
            self.resume = False
            self.kodi.play(name)

    def _on(self, name):
        # Deque old turn off TV events.
        deque(map(self.scheduler.cancel, self.scheduler.queue))

        if self.is_on:
            self._play(name)
        else:
            print("Turning on TV")

            self.kodi.login(name)

            if self.aux_on_command != "":
                try:
                    print("Running auxiliary 'on' command")
                    self._run_aux_command(self.aux_on_command)
                except CalledProcessError as e:
                    print("Failed to run auxiliary 'on' command: %s" % e)

            try:
                print("Sending CEC 'on' signal")
                self._cec_on()
                self.is_on = True
            except CalledProcessError as e:
                print("Failed to turn on TV: %s" % e)

        # Add new turn off TV event.
        self.scheduler.enter(self.off_timeout, 1, self._off)
        self.scheduler.enter(self.pause_timeout, 1, self._pause, argument=(name, ))

    def on(self, name="Unknown"):
        with self.lock:
            self._on(name)


class SleepDetector(Thread):
    def __init__(self, tv):
        self.tv = tv
        Thread.__init__(self)

    def _check_sleeping(self):
        r = requests.get("https://phewstoc.sladic.se/issleeping/")
        if r.status_code != 418:
            self.tv.on(name="Philip")

    def run(self):
        while True:
            self._check_sleeping()
            sleep(60*5)  # 5 minutes.


class FaceRecognizer(Thread):
    def __init__(self, people, camera_index, cores, tv):
        self.people = people
        self.camera_index = camera_index
        self.cores = cores
        self.tv = tv

        Thread.__init__(self)

    @staticmethod
    def _encode_face(path):
        hasher = hashlib.md5()
        blocksize = 65536

        # Compute image hash.
        with open(path, "rb") as f:
            b = f.read(blocksize)
            while len(b) > 0:
                hasher.update(b)
                b = f.read(blocksize)
        image_hash = hasher.hexdigest()

        image_cache = pathlib.Path(image_hash + ".npy")

        # Check if image encoding is cached on file system.
        if image_cache.exists():
            print("Loading image encoding cache at %s for file: %s" % (image_cache.name, path))
            return np.load(image_cache)

        # No cache found, encode image.
        else:
            print("Encoding image: %s" % path)
            image = face_recognition.load_image_file(path)

            face_encodings = face_recognition.face_encodings(image)
            if len(face_encodings) == 0:
                raise Exception("no face found in %s" % path)

            print("Successfully encoded: %s" % path)
            face_encoding = face_encodings[0]  # Use first face that was found.

            # Save image encoding to file system for re-use.
            np.save(image_hash, face_encoding)

            return face_encoding

    @staticmethod
    def _compare_face(known_face_names, known_face_encodings, face_encoding):
        # See if the face is a match for the known face(s)
        matches = face_recognition.compare_faces(known_face_encodings, face_encoding)
        name = "Unknown"
        authenticated = False

        # If a match was found in known_face_encodings, just use the first one.
        if True in matches:
            first_match_index = matches.index(True)
            name = known_face_names[first_match_index]
            authenticated = True

        return (name, authenticated)

    def run(self):
        cores = self.cores
        camera_index = self.camera_index
        people = self.people

        # The thread pool is used to run multiple encoders and recognition processes in parallel.
        # The init worker function (see lambda) forces the workers to ignore SIGINT, letting the main thread handling
        # the exit.
        thread_pool = mp.Pool(cores if cores != -1 else None, lambda: signal.signal(signal.SIGINT, signal.SIG_IGN))

        try:
            # Spawn one thread for each face encoder.
            encode_thread = thread_pool.map_async(FaceRecognizer._encode_face, [(p["image"]) for p in people])

            # While encoding the faces, open the video capture (webcam).
            video_capture = cv2.VideoCapture(camera_index)
            atexit.register(video_capture.release)  # Clean exit by releasing webcam on interrupt.

            known_face_names = [p["name"] for p in people]
            known_face_encodings = encode_thread.get(timeout=300)  # Collect result from threads, w/ timeout 5m.

            face_locations = []
            face_encodings = []

            print("Analyzing video capture at index %d" % camera_index)
            while video_capture.isOpened():
                # Grab a single frame of video
                ret, frame = video_capture.read()

                # Exit if empty frame (closed capture?).
                if frame is None:
                    break

                # Convert the image from BGR color (which OpenCV uses) to RGB color (which face_recognition uses).
                rgb_frame = frame[:, :, ::-1]

                # Find all the faces and face encodings in the current frame of video.
                face_locations = face_recognition.face_locations(rgb_frame)
                face_encodings = face_recognition.face_encodings(rgb_frame, face_locations)

                # Compare faces and display the results.
                for name, authenticated in thread_pool.starmap(FaceRecognizer._compare_face,
                                                               [(known_face_names, known_face_encodings, e)
                                                                for e in face_encodings]):

                    print("Found: %s" % name)
                    if authenticated:
                        self.tv.on(name=name)

            print("Capture #%d doesn't exist or was unexpectedly closed" % camera_index)

        except KeyboardInterrupt:
            thread_pool.terminate()
            thread_pool.join()


def main():
    parser = argparse.ArgumentParser(description="Recognize and identify faces.")
    parser.add_argument(
        "-i", "--camera", type=int, default=-1, metavar="N", help="Camera index, -1 picks the default camera")
    parser.add_argument(
        "-c",
        "--cores",
        type=int,
        default=-1,
        metavar="N",
        help="Number of cores to utilize, -1 will use all available cores")
    parser.add_argument(
        "-p",
        "--people",
        type=str,
        default="people.json",
        help="JSON encoded file with names and their corresponding image file path and password")
    parser.add_argument("-a", "--address", type=str, default="localhost", help="KODI control address")
    parser.add_argument("-t", "--tv", type=int, default=1, metavar="N", help="TV device index (CEC ID)")
    parser.add_argument(
        "-s",
        "--off-timeout",
        type=int,
        default=OFF_TIMEOUT_SECONDS,
        metavar="T",
        help="Timeout to send off signal if no face was recognized.")
    parser.add_argument(
        "-n",
        "--pause-timeout",
        type=int,
        default=PAUSE_TIMEOUT_SECONDS,
        metavar="T",
        help="Timeout to send pause signal if no face was recognized.")
    parser.add_argument("--aux-on-cmd", type=str, default="")
    parser.add_argument("--aux-off-cmd", type=str, default="")

    args = parser.parse_args()

    try:
        with open(args.people) as f:
            people = json.load(f)
    except FileNotFoundError:
        print("People data file: '%s' does not exit" % args.people)
        return

    # Create TV that is running KODI.
    kodi = KODI(people, args.address)
    tv = TV(
        args.tv,
        kodi,
        pause_timeout=args.pause_timeout,
        off_timeout=args.off_timeout,
        aux_on_command=args.aux_on_cmd,
        aux_off_command=args.aux_off_cmd)

    # Initialize feeders that will send detection events to the TV.
    sleep_detector = SleepDetector(people, tv)
    face_recognizer = FaceRecognizer(people, args.camera, args.cores, tv)

    # Start the feeders.
    sleep_detector.start()
    face_recognizer.start()

    # Wait for the threads to finish (which would probably be due to e.g. an keyboard interrupt)
    sleep_detector.join()
    face_recognizer.join()


if __name__ == "__main__":
    main()
