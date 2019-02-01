import atexit
import multiprocessing as mp
import signal
import sys

import cv2
import face_recognition


def encode_face(path):
    print("Encoding image: %s" % path)
    image = face_recognition.load_image_file(path)
    face_encoding = face_recognition.face_encodings(image)[0]
    print("Successfully encoded: %s" % path)

    return face_encoding


def compare_face(known_face_names, known_face_encodings, face_encoding):
    # See if the face is a match for the known face(s)
    matches = face_recognition.compare_faces(known_face_encodings, face_encoding)
    name = "Unknown"

    # If a match was found in known_face_encodings, just use the first one.
    if True in matches:
        first_match_index = matches.index(True)
        name = known_face_names[first_match_index]

    return name


def find_faces(people, camera_index):
    # The thread pool is used to run multiple encoders and recognition processes in parallel.
    thread_pool = mp.Pool(None, lambda: signal.signal(signal.SIGINT, signal.SIG_IGN))

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
        while True:
            # Grab a single frame of video
            ret, frame = video_capture.read()

            # Convert the image from BGR color (which OpenCV uses) to RGB color (which face_recognition uses).
            rgb_frame = frame[:, :, ::-1]

            # Find all the faces and face encodings in the current frame of video.
            face_locations = face_recognition.face_locations(rgb_frame)
            face_encodings = face_recognition.face_encodings(rgb_frame, face_locations)

            # Compare faces and display the results.
            for name in thread_pool.starmap(
                    compare_face,
                    [(known_face_names, known_face_encodings, e) for e in face_encodings]):
                print(name)

    except KeyboardInterrupt:
        thread_pool.terminate()
        thread_pool.join()


find_faces([
    {
        "name": "William",
        "path": "william.jpg"
    },
    {
        "name": "Philip",
        "path": "philip.jpg"
    },
    {
        "name": "Rebecka",
        "path": "rebecka.jpg"
    },
    {
        "name": "Obama",
        "path": "obama.jpg"
    },
    {
        "name": "Edvin",
        "path": "edvin.jpg"
    },
    {
        "name": "Biden",
        "path": "biden.jpg"
    },
    {
        "name": "Hampus",
        "path": "hampus.jpg"
    }
], int(sys.argv[1]))
