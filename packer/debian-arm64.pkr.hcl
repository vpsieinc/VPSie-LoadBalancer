source "qemu" "debian-arm64" {
  iso_url          = "https://cdimage.debian.org/debian-cd/current/arm64/iso-cd/debian-${var.debian_version}-arm64-netinst.iso"
  iso_checksum     = "file:https://cdimage.debian.org/debian-cd/current/arm64/iso-cd/SHA256SUMS"
  output_directory = "output/arm64"
  vm_name          = "vpsie-lb-debian-12-arm64-${var.version}.qcow2"
  format           = "qcow2"

  disk_size        = "10G"
  disk_compression = true

  memory           = 2048
  cpus             = 2

  headless         = true

  qemu_binary      = "qemu-system-aarch64"
  machine_type     = "virt"
  cpu_model        = "cortex-a57"

  qemuargs = [
    ["-bios", "/usr/share/qemu-efi-aarch64/QEMU_EFI.fd"],
    ["-boot", "strict=on"]
  ]

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
  sources = ["source.qemu.debian-arm64"]

  provisioner "shell" {
    scripts = [
      "scripts/provision.sh",
      "scripts/install-envoy.sh",
      "scripts/install-agent.sh",
      "scripts/setup-systemd.sh",
      "scripts/cleanup.sh"
    ]
  }

  post-processor "checksum" {
    checksum_types = ["sha256"]
    output         = "output/arm64/vpsie-lb-debian-12-arm64-${var.version}.checksum"
  }
}
