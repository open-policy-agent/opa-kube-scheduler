FROM scratch
MAINTAINER Torin Sandall <torinsandall@gmail.com>
ADD rego-scheduler /rego-scheduler
ENTRYPOINT ["/rego-scheduler"]

