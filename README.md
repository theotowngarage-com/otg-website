
![Build Status](https://github.com/theotowngarage-com/otg-website/actions/workflows/hugo.yml/badge.svg)

---

Static landing page made with [Hugo] using GitHub Pages.

Learn more about GitLab Pages at https://pages.gitlab.io and the official
documentation https://docs.gitlab.com/ce/user/project/pages/.

## GitHub CI

This project's static Pages are built & deployed by [GitHub CI (Actions)][actions], following the steps defined in [`.github/workflows/hugo.yml`](.github/workflows/hugo.yml).

It can _also_ be built using [GitLab CI][ci], following the steps
defined in [`.gitlab-ci.yml`](.gitlab-ci.yml).

## Deployment

Deployment always happen on the latest commit on the master branch.

## Building locally

To work locally with this project, you'll have to follow the steps below:

1. Fork, clone or download this project
1. Install Hugo (see below)
1. Preview your project: `hugo server`
1. Add content. The local preview is rebuilt and refreshed in your browser.
1. To generate the final static website: `hugo` (optional)

Read more at Hugo's [documentation][].

### Install Hugo
<a name="install_hugo">

Install the [extended edition](https://gohugo.io/categories/installation/) of Hugo

To confirm that it's correctly installed, type `hugo version`.

### Preview your site

If you clone or download this project to your local computer and run `hugo server`,
your site can be accessed under `localhost:1313/`.

[actions]: https://docs.github.com/en/actions
[ci]: https://about.gitlab.com/gitlab-ci/
[hugo]: https://gohugo.io
[documentation]: https://gohugo.io/overview/introduction/