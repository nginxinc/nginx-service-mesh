# Contributing Guidelines

The following is a set of guidelines for contributing to NGINX Service Mesh. We really appreciate that you are considering contributing!

#### Table Of Contents

[Ask a Question](#ask-a-question)

[Getting Started](#getting-started)

[Contributing](#contributing)

[Style Guides](#style-guides)
  * [Git Style Guide](#git-style-guide)
  * [Go Style Guide](#go-style-guide)

[Code of Conduct](CODE_OF_CONDUCT.md)

[Contributor License Agreement](#contributor-license-agreement)

## Ask a Question

Questions can be asked via [Issues](https://github.com/nginxinc/nginx-service-mesh/issues), [Discussions](https://github.com/nginxinc/nginx-service-mesh/discussions), or on the [NGINX Community Slack](https://nginxcommunity.slack.com/channels/nginx-service-mesh) in the `#nginx-service-mesh` channel.

## Getting Started

Follow our [Getting Started](https://docs.nginx.com/nginx-service-mesh/get-started/) guides to get NGINX Service Mesh up and running.

### Project Structure

* NGINX Service Mesh is written in Go and uses the NGINX Plus software as the data plane.
* The project follows a standard Go project layout
    * The main code is found at `cmd/`
    * The internal code is found at `internal/`
    * External APIs, clients, and SDKs can be found under `pkg/`
    * Example configurations and applications can be found under `examples/`
* We use [Go Modules](https://github.com/golang/go/wiki/Modules) for managing dependencies.
* We use [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/) for our BDD style unit tests.

## Contributing

### Report a Bug

To report a bug, open an issue on GitHub with the label `bug` using the available bug report issue template. Please ensure the issue has not already been reported.

### Suggest an Enhancement

To suggest an enhancement, please create an issue on GitHub with the label `enhancement` using the available feature issue template.

### Open a Pull Request

* Fork the repo, create a branch, submit a PR when your changes are tested and ready for review
* Fill in [our pull request template](.github/PULL_REQUEST_TEMPLATE.md)

> **Note**
>
> If you’d like to implement a new feature, please consider creating a feature request issue first to start a discussion about the feature.

### Issue lifecycle

* When an issue or PR is created, it will be triaged by the core development team and assigned a label to indicate the type of issue it is (bug, feature request, etc) and to determine the milestone. Please see the [Issue Lifecycle](ISSUE_LIFECYCLE.md) document for more information.

## Style Guides

### Git Style Guide

* Keep a clean, concise and meaningful git commit history on your branch, rebasing locally and squashing before submitting a PR
* Follow the guidelines of writing a good commit message as described [here](https://chris.beams.io/posts/git-commit/) and summarized in the next few points
    * In the subject line, use the present tense ("Add feature" not "Added feature")
    * In the subject line, use the imperative mood ("Move cursor to..." not "Moves cursor to...")
    * Limit the subject line to 72 characters or less
    * Reference issues and pull requests liberally after the subject line
    * Add more detailed description in the body of the git message (`git commit -a` to give you more space and time in your text editor to write a good message instead of `git commit -am`)

### Go Style Guide

* Run `make format` over your code to automatically resolve a lot of style issues. Most editors support this running automatically when saving a code file.
* Run `make lint` to catch any other issues.
* Follow this guide on some good practice and idioms for Go -  https://github.com/golang/go/wiki/CodeReviewComments

## Contributor License Agreement

Individuals or business entities who contribute to this project must have completed and submitted the [F5® Contributor License Agreement](F5ContributorLicenseAgreement.pdf) prior to their code submission being included in this project.
To submit, please print out the [F5® Contributor License Agreement](F5ContributorLicenseAgreement.pdf), fill in the required sections, sign, scan, and send executed CLA to kubernetes@nginx.com.
Please include your github handle in the CLA email.
