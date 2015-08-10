FROM lampdb/lamp-devbase:latest

MAINTAINER Tobias Schottdorf <tobias.schottdorf@gmail.com>

ENV ROACHPATH /go/src/github.com/lampdb

# Copy the contents of the lamp source directory to the image.
# Any changes which have been made to the source directory will cause
# the docker image to be rebuilt starting at this cached step.
ADD . ${ROACHPATH}/lamp/
RUN ln -s ${ROACHPATH}/lamp/build/devbase/lamp.sh ${ROACHPATH}/lamp/lamp.sh

# Build the lamp executable.
RUN cd -P ${ROACHPATH}/lamp && make build

# Expose the http status port.
EXPOSE 8080

# This is the command to run when this image is launched as a container.
# Environment variable expansion doesn't seem to work here.
ENTRYPOINT ["/go/src/github.com/lampdb/lamp/lamp.sh"]

# These are default arguments to the lamp binary.
CMD ["--help"]
