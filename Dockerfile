FROM scratch
MAINTAINER Torin Sandall <torinsandall@gmail.com>
ADD rego-scheduler /rego-scheduler
CMD ["/rego-scheduler", "--v=2", "--logtostderr=1"]
