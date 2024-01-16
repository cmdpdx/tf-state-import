## tf-state-import

A tool for importing resources from a given Terraform state file. Outputs a list of `terraform state rm`
and `terraform state import` commands to refresh a state file. Helpful when transitioning between
incompatible versions of a provider.

Resource dependencies are considered when compiling the `rm` and `import` statements. Resources
are removed from most dependent to least dependant resource, and imported in the opposite order.

### Usage

```
$ go install .

# Run with no flags to look in the current directory for terraform.tfstate
# and output the commands to stdout.
$ tf-state-import
terraform state rm resource_foo.name
terraform state rm resource_bar.name
...
terraform state import resource_bar.name
terraform state import resource_foo.name

# Optionally pass the location of a statefile and an output script file.
$ tf-state-import --tfstate=/path/to/statefile.tfstate --out=/path/to/script/import.sh
```
