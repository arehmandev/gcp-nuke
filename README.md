# Gcp-Nuke

![Nukedown](https://github.com/arehmandev/gcp-nuke/nuclear.png)

Beware, only members and personnel of Area 51 are allowed past this point.

## Background

Inspired by aws-nuke.

This tool was created out of my personal frustration with cleaning up GCP projects. 

But why?

Many reasons:

1) The behaviour of gcloud projects delete is to disable a project - pending a 30 day wait time for any resource removal. Unfortunately, this behaviour breaks SharedVPCs - service project deletion can cause "ghost subnets" on the Host project, - yes, you end up with an undeletable subnets due to VM resources. Google support's solution? Well ofcourse, "just don't do it" - https://cloud.google.com/vpc/docs/deprovisioning-shared-vpc.

Additionally, I've found Terraform destroy of some of my colleagues' wizard level terraform modules fail occasionally, so it's always neat to see what's not been deleted via a dryrun.

## Usage

```
NAME:
   gcp-nuke - A new cli application

USAGE:
   e.g. gcp-nuke --project test-nuke-262510 --dryrun

VERSION:
   v0.1.0

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --project value   GCP project id to nuke (default: "project")
   --dryrun          Perform a dryrun instead (default: false)
   --timeout value   Timeout for removal of a single resource in seconds (default: 400)
   --polltime value  Time for polling resource deletion status in seconds (default: 10)
   --help, -h        show help (default: false)
   --version, -v     print the version (default: false)
```

## Roadmap

- Add unit tests and create a pipeline for robust integration test cases
- DRY - unfortunately due to the lack of generics in Go, I feel much of the code feels replicated among resources, lets come up with a solution
- More reliable Dependencies and errors - Currently each resource can supply a list of dependent resources to remove first, however this always work as planned,