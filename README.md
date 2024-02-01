## Steps

1.  Upload image to Prism Central

    <https://cloud-images.ubuntu.com/minimal/releases/jammy/release-20240125/ubuntu-22.04-minimal-cloudimg-amd64.img>

1.  Create cloud-init user-data.

    - Write systemd service files
    - Run command to download capx executable
    - Run command to start capx systemd service

1.  Create VM

    - Use image as disk
    - Passes cloud-init user-data with kubeconfig and Prism Central configuration. Note: This is information is sensitive.
