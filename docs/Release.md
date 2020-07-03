# How to publish a release

Create a new release in just a few steps:

1. Make sure you're on the master branch, and pull the latest changes.

    ```
    git checkout master
    git pull origin master
    ```

2. Create a new branch in the shape of `master#release#v<MAJOR.MINOR.PATCH>` and push it to the remote repository

    For example, for the `v1.0.0` version:

    ```
    git checkout -b master#release#v1.0.0
    git push origin master#release#v1.0.0
    ```

3. You will be assigned to a Pull Request in the GitHub repository, after the GitHub workflow finishes running. You can check its status [here](https://github.com/giantswarm/kubectl-gs/actions), by looking at the `Create Release PR` workflow.

4. Ask for a Pull Request review from your team or the project owners. After getting it approved and ready to go, squash and merge it.

5. The PR merge will trigger a GitHub workflow, which will create a tag, a new release, package the build artifacts and attach them to the newly created release.

6. [Edit your newly created release](https://github.com/giantswarm/kubectl-gs/releases) and add release notes. 

7. Update the `krew-index` repository automatically [by approving](https://app.circleci.com/pipelines/github/giantswarm/kubectl-gs) the `update-krew` CircleCI workflow. A Pull Request to the `krew-index` repository will be created and merged automatically (by robots 🤖).

8. 🎉 Celebrate by announcing the fresh release on Slack! 
