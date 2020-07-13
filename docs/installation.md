# Installation

`kubectl-gs` is the Giant Swarm plug-in for `kubectl` with the official plug-in name `gs`.

The simplest way to manage `kubectl` plug-ins accross platforms is using [Krew](https://krew.sigs.k8s.io/). If you don't have Krew installed, check the [Krew installation docs](https://krew.sigs.k8s.io/docs/user-guide/setup/install/) on how to get it installed.

Further down you will also find instructions on installing the plug-in [without Krew](#without-krew).

## Using Krew

To install the `gs` plug-in, simply execute this command:

```nohighlight
kubectl krew install gs
```

We highly recommend to set up the `kgs` shorthand as well:

```nohighlight
alias kgs="kubectl gs"
```

(Best add this to your shell profile or config file.)

Lastly, let's check that the plugin is working as it's supposed to.

```nohighlight
kgs
```

You should see information regarding the commands available.

To upgrade to the latest version of the plug-in, use this command:

```nohighlight
krew upgrade gs
```

## Without Krew

The platform-agnostic description:

1. Download the [latest release](https://github.com/giantswarm/kubectl-gs/releases/latest) archive for your platform
2. Unpack the archive
3. Copy the executable to a location included in your `$PATH`
4. Create an alias `kgs` for `kubectl gs`
5. Check if it's working by executing `kgs`

### Linux

```bash
# Determine the latest version
VERSION=$(curl -I -sS https://github.com/giantswarm/kubectl-gs/releases/latest|grep 'location:'|awk -F '/' '{print $NF}'|tr -d '\n'|tr -d '\r')

# Download
wget https://github.com/giantswarm/kubectl-gs/releases/download/${VERSION}/kubectl-gs-${VERSION}-linux-amd64.tar.gz

# Unpack
tar xzf kubectl-gs-${VERSION}-linux-amd64.tar.gz

# Copy to a dir in $PATH
cp kubectl-gs-${VERSION}-linux-amd64/kubectl-gs /usr/local/bin/

# Set up alias
alias kgs="kubectl gs"

# Check
kgs
```

### Mac OS

```bash
# Determine the latest version
VERSION=$(curl -I -sS https://github.com/giantswarm/kubectl-gs/releases/latest|grep 'location:'|awk -F '/' '{print $NF}'|tr -d '\n'|tr -d '\r')

# Download
wget https://github.com/giantswarm/kubectl-gs/releases/download/${VERSION}/kubectl-gs-${VERSION}-darwin-amd64.tar.gz

# Unpack
tar xzf kubectl-gs-${VERSION}-darwin-amd64.tar.gz

# Copy to a dir in $PATH
cp kubectl-gs-${VERSION}-darwin-amd64/kubectl-gs /usr/local/bin/

# Set up alias
alias kgs="kubectl gs"

# Check
kgs
```
