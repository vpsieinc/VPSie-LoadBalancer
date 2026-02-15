source "qemu" "debian-amd64" {
  iso_url          = "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-${var.debian_version}-amd64-netinst.iso"
  iso_checksum     = "file:https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/SHA256SUMS"
  output_directory = "output/amd64"
  vm_name          = "vpsie-lb-debian-13-amd64-${var.version}.qcow2"
  format           = "qcow2"
  accelerator      = "kvm"

  disk_size        = "10G"
  disk_compression = true

  memory           = 2048
  cpus             = 2

  headless         = true

  http_directory   = "http"

  boot_wait        = "5s"
  boot_command = [
    "<esc><wait>",
    "install <wait>",
    "preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg <wait>",
    "debian-installer=en_US.UTF-8 <wait>",
    "auto <wait>",
    "locale=en_US.UTF-8 <wait>",
    "kbd-chooser/method=us <wait>",
    "keyboard-configuration/xkb-keymap=us <wait>",
    "netcfg/get_hostname=vpsie-lb <wait>",
    "netcfg/get_domain=local <wait>",
    "fb=false <wait>",
    "debconf/frontend=noninteractive <wait>",
    "console-setup/ask_detect=false <wait>",
    "console-keymaps-at/keymap=us <wait>",
    "<enter><wait>"
  ]

  ssh_username     = "root"
  ssh_password     = "vpsie"
  ssh_timeout      = "30m"

  shutdown_command = "echo 'vpsie' | sudo -S shutdown -P now"
}

build {
  sources = ["source.qemu.debian-amd64"]

  provisioner "shell" {
    scripts = [
      "scripts/provision.sh",
      "scripts/install-envoy.sh",
      "scripts/install-agent.sh",
      "scripts/setup-systemd.sh",
      "scripts/cleanup.sh"
    ]
  }

  # Lock build-time passwords - these are only used for Packer SSH access during build
  # and must be locked before the image is distributed
  provisioner "shell" {
    inline = [
      "passwd -l root",
      "passwd -l vpsie",
      "echo 'Build-time passwords locked. Use cloud-init or VPSie API for access provisioning.'"
    ]
  }

  post-processor "checksum" {
    checksum_types = ["sha256"]
    output         = "output/amd64/vpsie-lb-debian-13-amd64-${var.version}.checksum"
  }
}
