# Gotiator

A tiny API Gateway based on [JWTs](https://jwt.io/).

Gotiator can handle simple API proxying with signing for single page apps that already use JWTs for authentication.

Gotiator Proxy is released under the [MIT License](LICENSE).
Please make sure you understand its [implications and guarantees](https://writing.kemitchell.com/2016/09/21/MIT-License-Line-by-Line.html).

## Installing

```
go get github.com/netlify/gotiator
gotiator serve
```

## Configuration

Settings can be set either by creating a `config.json` or setting `NETLIFY_` prefixed environment
variables. IE.:

```json
{
  "jwt": {
    "secret": "2134"
  }
}
```

Is the same as:

```
GOTIATOR_JWT_SECRET=2134 gotiator serve
```

You must set your JWT secret (and we strongly recommend doing this with an environment variable)
to match the JWT issuer (like [Auth0](https://auth0.com)) or [netlify-auth](https://github.com/netlify/netlify-auth).

You configure API proxying from the config.json:

```
{
  "apis": [
    {"name": "github", "url": "https://api.github.com/repos/netlify/gotiator", "roles": ["contributor"]}
  ]
}
```

To sign outgoing requests with a Bearer token, you must set an environment variable with the token,
based on the name of the API. If the API is called `github`, you must set:

```
NETLIFY_API_GITHUB=1234
```

The `roles` property specifies which roles should have access to the API. Roles should be encoded in the
JWT claims under `app_metadata.roles`. Any request with a correctly signed JWT that includes one of the
roles in it's `app_metadata` will be allowed to make requests to the API signed with your token via
`/:api_name`.

With the above example, a user with a JWT proving the claim that she has the role "contributor", can
send signed requests to GitHub's API scoped to this repo, via:

```
GET|POST|DELETE|PATCH /github
```
