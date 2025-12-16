packer {
  required_plugins {
    qemu = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

variable "version" {
  type    = string
  default = "1.0.0"
}

variable "debian_version" {
  type    = string
  default = "13.2.0"
}
