# most-popular-committer
[![codecov](https://codecov.io/gh/RafalKorepta/most-popular-committer/branch/develop/graph/badge.svg)](https://codecov.io/gh/RafalKorepta/most-popular-committer)
[![Build Status](https://travis-ci.org/RafalKorepta/most-popular-committer.svg?branch=develop)](https://travis-ci.org/RafalKorepta/most-popular-committer)

# Build

You can simply run:
```bash
make
```

# Run

You can change create new configuration file or change existing `.most-popular-committer.yml`.
After that please run command:
```bash
make run
```

## Endpoints

- http://ip:9091/v1alpha1/committer?language=java
- http://ip:9091/swagger.json
- http://ip:9091/swagger-ui
- http://ip:9091/metrics

# Rate limiting

The grpc rate limiting is not available yet. 
The PR is open https://github.com/grpc-ecosystem/go-grpc-middleware/pull/181, but is not merged.
Current implementation of server uses this implementation as it is copied to `ratelimit` package.

## Test

To test the behavior of rate limiting run `make && make run`. 
In the second console run `./run-concurent-request.sh`.
