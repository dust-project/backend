import os
import os.path
import subprocess

generated_react_dir = ""

if not os.path.exists("./dust"):

    os.mkdir("dust")
    os.chdir("dust")

    if subprocess.call(["git", "clone", "https://__token__@github.com/dust-project/backend"]) != 0:
        print("Authentication error")
        exit()

    if subprocess.call(["git", "clone", "https://__token__@github.com/dust-project/frontend"]) != 0:
        print("Authentication error")
        exit()

os.chdir("./dust/backend")
if subprocess.call(["go", "build"]) != 0:
    print("Failed to build backend server")
    exit()

subprocess.call(["./dust",f"staticdir={generated_react_dir}"])
