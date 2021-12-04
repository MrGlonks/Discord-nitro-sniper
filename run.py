import subprocess

try:
    while True:
        cmd = subprocess.run("GlockSniper")
        if cmd.returncode == 42:
            cmd = subprocess.run("GlockSniper")
        else:
            break
except:
    pass
