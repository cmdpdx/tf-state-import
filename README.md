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
$ tf-state-import
terraform state rm resource_foo.name
terraform state rm resource_bar.name
...
terraform state import resource_bar.name bar_id
terraform state import resource_foo.name foo_id

# Generate terraform import blocks 
$ tf-state-import --tfstate=/path/to/statefile.tfstate --format=block
import {
  to = resource_foo.name
  id = foo_id
}
import {
  to = resource_bar.name
  id = bar_id
}
```
