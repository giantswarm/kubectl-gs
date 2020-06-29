# Installation

The recommended way to install the `kubectl-gs`, the `gs` plugin for `kubectl`, and for keeping it up-to-date, is using [Krew](https://krew.sigs.k8s.io/). However here we also explain how to do it without Krew.

## Using Krew

If you don't have Krew installed, check the upstream [documentation](https://krew.sigs.k8s.io/docs/user-guide/setup/install/) on how to get it installed.

With [Krew](https://krew.sigs.k8s.io/):

```nohighlight
kubectl krew update
kubectl krew install gs
```

Now you have the plugin installed.

Let's make sure it is easy to invoke via the `kgs` alias. That's what we use in our documentation, too, to make things snappy.

```nohighlight
alias kgs="kubectl gs"
```

(Best add this to your shell profile or config file.)

Lastly, let's check that the plugin is working as it's supposed to.

```nohighlight
kgs info
```

You should see some friendly information.

## Not using Krew

The platform-agnostic description:

1. Download the [latest release](https://github.com/giantswarm/kubectl-gs/releases/latest) archive for your platform
2. Unpack the archive
3. Copy the executable to a location included in your `$PATH`
4. Create an alias `kgs` for `kubectl gs`
5. Check it's working by executing `kgs info`

### Linux 64Bit

For version 0.5.0:

```nohighlight
wget https://github.com/giantswarm/kubectl-gs/releases/download/v0.5.0/kubectl-gs_0.5.0_linux_amd64.tar.gz
tar xzf kubectl-gs_0.5.0_linux_amd64.tar.gz
cp ./kubectl-gs /usr/local/bin/
alias kgs="kubectl gs"
kgs info
```

### Mac OS

For version 0.5.0:

```nohighlight
wget https://github.com/giantswarm/kubectl-gs/releases/download/v0.5.0/kubectl-gs_0.5.0_darwin_amd64.tar.gz
tar xzf kubectl-gs_0.5.0_darwin_amd64.tar.gz
cp ./kubectl-gs /usr/local/bin/
alias kgs="kubectl gs"
kgs info
```
