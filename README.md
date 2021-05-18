# Skate

A personal key value store. 🛼

# What is it?

Skate is a key value store that you can use to save and retrieve valuable bits
of data. It's fully encrypted, backed up to the cloud (that you can self-host
if you want) and linkable to any machine you use.

## Examples

### Keep secrets out of your .bashrc

```
skate set gh_token GITHUB_TOKEN
echo "export GITHUB_TOKEN=$(skate get gh_token)" >> ~/.bashrc
```

### Save some passwords

```
skate set github@password.db PASSWORD
skate get github@password.db
```

### Easily store data in scripts

```
#!/bin/bash

skate set saved_stuff $1
echo "We just saved $(skate get saved_stuff)"
```

# Installation

Use your fave package manager:

```
```

# Self-hosting

Skate is backed by the Charm Cloud. By default it will use the Charm hosted
cloud, but if you want to self-host you can download
[charm](https://github.com/charmbracelet/charm) and run you own cloud with
`charm serve`. Then set the `CHARM_HOST` environment variable to the hostname
of your Charm Cloud server.

# License

[MIT](https://github.com/charmbracelet/skate/raw/master/LICENSE)

<a href="https://charm.sh/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge-unrounded.jpg" width="400"></a>

Charm热爱开源! / Charm loves open source!
