---
title: Base image
sidebar: reference
permalink: reference/build/base_image.html
author: Alexey Igrychev <alexey.igrychev@flant.com>
---

Here's a minimal `dappfile.yml`. It describes a _dimg_ named `example` that is based on a _base image_ named `alpine`:

```yaml
dimg: example
from: alpine
```

_Base image_ can be declared with `from`, `fromDimg` or `fromDimgArtifact` directives.   

## from and fromCacheVersion

The `from` directive sets the name and tag of a _base image_ to build _from stage_. If absent, tag defaults to `latest`.

```yaml
from: <image>[:<tag>]
```

On _from stage_, _from image_ is pulled from a repository if it doesn't exist in local docker, and saved in the [_stages cache_]({{ site.baseurl }}/reference/build/stages.html).

Pay attention, _from stage_ depends on a variable of _from directive_, not on _from image_ digest or something else. 

If you have, e.g., _from image_ `alpine:latest`, and need to rebuild _dimg_ with the actual latest alpine image you should set _fromCacheVersion_ (`fromCacheVersion: <arbitrary string>`) and delete existent _from image_ from local docker or manually pull it. 

If you want always build dimg with actual latest _from image_ you should automate this process outside of dapp and use a particular environment variable for `fromCacheVersion` (e.g., the variable with _from image_ digest).

{% raw %}
```yaml
from: "alpine:latest"
fromCacheVersion: {{ env "FROM_IMAGE_DIGEST" }}
```
{% endraw %}

## fromDimg and fromDimgArtifact

Besides using docker images from a repository, _base image_ can refer to _dimg_ or [_artifact_]({{ site.baseurl }}/reference/build/artifact_directive.html), described in the same `dappfile.yml`.

```yaml
fromDimg: <dimg name>
fromDimgArtifact: <artifact name>
```

If an intermediate image is specific to a particular application,
it is reasonable to store its description with _dimgs_ and _artifacts_ which are used it as opposed to storing _base image_ in a registry.

Also, this method can be helpful if the stages of _stage conveyor_ are not enough for building an image. You can design your _stage conveyor_.

<a href="https://docs.google.com/drawings/d/e/2PACX-1vQFIMrYCWTPiLImK3QSDl-b_Ch_mEIy8zCK-S_v6oeeuz6UXISDxXVcXqHO2wH1Oa9Y9RJIQU33rRfE/pub?w=1629&amp;h=1435" data-featherlight="image">
<img src="https://docs.google.com/drawings/d/e/2PACX-1vQFIMrYCWTPiLImK3QSDl-b_Ch_mEIy8zCK-S_v6oeeuz6UXISDxXVcXqHO2wH1Oa9Y9RJIQU33rRfE/pub?w=815&amp;h=718">
</a>