#!/usr/bin/env python3

import argparse
import atexit
import json
import multiprocessing as mp
import signal

import cv2
import face_recognition


def encode_face(path):
    print("Encoding image: %s" % path)
    image = face_recognition.load_image_file(path)

    face_encodings = face_recognition.face_encodings(image)
    if len(face_encodings) == 0:
        raise Exception("no face found in %s" % path)

    print("Successfully encoded: %s" % path)
    return face_encodings[0]  # Use first face that was found.


def compare_face(known_face_names, known_face_encodings, face_encoding):
    # See if the face is a match for the known face(s)
    matches = face_recognition.compare_faces(known_face_encodings, face_encoding)
    name = "Unknown"

    # If a match was found in known_face_encodings, just use the first one.
    if True in matches:
        first_match_index = matches.index(True)
        name = known_face_names[first_match_index]

    return name


def find_faces(people, camera_index, cores):
    # The thread pool is used to run multiple encoders and recognition processes in parallel.
    # The init worker function (see lambda) forces the workers to ignore SIGINT, letting the main thread handling the
    # exit.
    thread_pool: mp.Pool = mp.Pool(cores if cores != -1 else None, lambda: signal.signal(signal.SIGINT, signal.SIG_IGN))

    try:
        # Spawn one thread for each face encoder.
        encode_thread = thread_pool.map_async(encode_face, [(p['path']) for p in people])

        # While encoding the faces, open the video capture (webcam).
        video_capture = cv2.VideoCapture(camera_index)
        atexit.register(video_capture.release)  # Clean exit by releasing webcam on interrupt.

        known_face_names = [p["name"] for p in people]
        known_face_encodings = encode_thread.get(timeout=120)  # Collect result from threads, w/ timeout 2m.

        face_locations = []
        face_encodings = []
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
            for name in thread_pool.starmap(
                    compare_face,
                    [(known_face_names, known_face_encodings, e) for e in face_encodings]):
                print("Found: %s" % name)

        print("Capture #%d doesn't exist or was unexpectedly closed" % camera_index)

    except KeyboardInterrupt:
        thread_pool.terminate()
        thread_pool.join()


def main():
    parser = argparse.ArgumentParser(description="Recognize and identify faces.")
    parser.add_argument("-i", "--camera", type=int, default=-1, metavar="N",
                        help="Camera index, -1 picks the default camera")
    parser.add_argument("-c", "--cores", type=int, default=-1, metavar="N",
                        help="Number of cores to utilize, -1 will use all available cores")
    parser.add_argument("-p", "--people", type=str, default="people.json",
                        help="JSON encoded file with names and their corresponding image file path")

    args = parser.parse_args()

    try:
        with open(args.people) as f:
            people = json.load(f)
    except FileNotFoundError:
        print("People data file: '%s' does not exit" % args.people)
        return

    find_faces(people, args.camera, args.cores)


if __name__ == "__main__":
    main()
