FROM scratch
MAINTAINER Torin Sandall <torinsandall@gmail.com>
ADD opa-kube-scheduler_linux_amd64 /opa-kube-scheduler
CMD ["/opa-kube-scheduler", "--v=2", "--logtostderr=1"]
