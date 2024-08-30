# Development

Start by installing [`devbox`](https://www.jetify.com/devbox/docs/installing_devbox/) and optionally `direnv` (you could [use devbox to install it globally](https://www.jetify.com/devbox/docs/devbox_global/) by using `devbox global add direnv`).

This should allow to use `make`.

Check out the targets available in the Make file by running `make help`.

## Releasing

1. Trigger a [release build workflow run](https://github.com/cloudfoundry/app-autoscaler-cli-plugin/actions/workflows/release.yml). You need to manually determine the correct semantic version. This could be automated in the future.
2. In the output of the release job you triggered, navigate to the output of the step "Update plugin repo". At the bottom you will find a line similar to:
   > remote: Create a pull request for 'bump-app-autoscaler-cli-plugin-to-vX.X.X' on GitHub by visiting:

   Click the link in the following line.
   This will open a PR with a template. Remove the template text after the first sentence and optionally add some notes. Create the PR and make sure it gets merged eventually.
3. Once it is merged, after up to 24 hours later the release should appear at the top of [plugins.cloudfoundry.org](https://plugins.cloudfoundry.org/). Once this is the case it can be consumed using the regular install command listed in the [main README.md](../README.md).
