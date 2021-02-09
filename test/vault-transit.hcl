listener "tcp" {
  address     = "127.0.0.1:8100"
  tls_disable = true
}

storage "inmem" {}

seal "transit" {
  address         = "http://127.0.0.1:8200"
  token           = "test"
  disable_renewal = "true"
  key_name        = "autounseal"
  mount_path      = "transit/"
  tls_skip_verify = true
}
