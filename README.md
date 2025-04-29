
![Build Status](https://github.com/theotowngarage-com/otg-website/actions/workflows/hugo.yml/badge.svg)

---

Static landing page made with [Hugo] using GitHub Pages.

## GitHub CI

This project's static Pages are built & deployed by [GitHub CI (Actions)][actions], following the steps defined in [`hugo.yml`].

It can _also_ be built using [GitLab CI][ci], following the steps
defined in [`.gitlab-ci.yml`](.gitlab-ci.yml).

## Deployment

Deployment always happen on the latest commit on the master branch.

## Building locally

To work locally with this project, you'll have to follow the steps below:

1. Install Hugo (see below)
1. Fork, clone or download this project
1. Go to the root folder, and run `hugo server`
1. Preview the website under `localhost:1313/`
1. Add content, modify the files. The local preview is rebuilt and refreshed in your browser.
1. commit and push your changes, submit a PR etc :D
1. (optional) To generate the final static website, simply run `hugo` (see [`hugo.yml`] for deployment optimisations)

Read more at Hugo's [documentation].

### Install Hugo

[Install the extended edition] of Hugo (you don't need the extended/deploy edition)

To confirm that it's correctly installed, type `hugo version` (only the `+extended` matters).

```
hugo v0.136.5+extended linux/amd64 BuildDate=unknown VendorInfo=nixpkgs
```

### Preview your site

Go to the root folder of the project, run `hugo server`,
and access the website under `localhost:1313/`, or wherever tells you it is.

[`hugo.yml`]: .github/workflows/hugo.yml
[actions]: https://docs.github.com/en/actions
[ci]: https://about.gitlab.com/gitlab-ci/
[hugo]: https://gohugo.io
[documentation]: https://gohugo.io/overview/introduction/
[Install the extended edition]: https://gohugo.io/installation/