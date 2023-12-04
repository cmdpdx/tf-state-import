## tf-state-import

A tool for importing resources from a given Terraform state file.

### Usage

```
$ go run . --tfstate=/path/to/statefile.tfstate --out=/path/to/script/import.sh

$ /path/to/script/import.sh
terraform import resource id
...
```
