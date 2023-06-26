save_method "file" {
  path = "vault-init.json"
}

save_method "kube_secret" {
  name = "vault-init"
  key = "init.json"
}
