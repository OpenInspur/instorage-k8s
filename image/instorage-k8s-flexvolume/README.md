# Steps to use the DaemonSet deployment method

1. Copy the Flexvolume driver to `drivers` directory. To get a basic example running, copy the `dummy` driver from the parent directory.
2. If you'd like to just get a basic example running, you could skip this step. Otherwise, change the places marked with `TODO` in all files.
3. Build the deployment Docker image and upload to your container registry.
4. Create the DaemonSet.
