terraform {
  required_providers {
    tls = {
      source = "hashicorp/tls"
      version = "4.0.6"
    }
        local = {
      source = "hashicorp/local"
      version = "2.5.2"
    }
  }
}
