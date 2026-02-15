source "qemu" "debian-amd64" {
  iso_url          = "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-${var.debian_version}-amd64-netinst.iso"
  iso_checksum     = "sha512:1ada40e4c938528dd8e6b9c88c19b978a0f8e2a6757b9cf634987012d37ec98503ebf3e05acbae9be4c0ec00b52e8852106de1bda93a2399d125facea45400f8"
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

  # Lock build-time passwords before image ships
  provisioner "shell" {
    inline = [
      "# Lock passwords set during preseed - they are only needed for packer SSH access",
      "passwd -l root",
      "passwd -l vpsie"
    ]
  }

  post-processor "checksum" {
    checksum_types = ["sha256"]
    output         = "output/amd64/vpsie-lb-debian-13-amd64-${var.version}.checksum"
  }
}
