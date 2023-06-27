save_method "file" {
  path      = "vault-init.json"
  overwrite = true
}

save_method "kube_secret" {
  name      = "vault-init"
  key       = "init.json"
  overwrite = true
  labels = {
    "foo" = "bar"
  }
  annotations = {
    "myanno.pbar.me" = "annotation is cool"
  }
}
