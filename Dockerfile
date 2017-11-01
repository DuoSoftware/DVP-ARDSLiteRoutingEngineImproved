# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang
ARG MAJOR_VER
# Copy the local package files to the container's workspace.
#ADD . /go/src/github.com/golang/example/outyet
#RUN go get github.com/DuoSoftware/DVP-ARDSLiteRoutingEngineImproved/ArdsLiteRoutingEngineImproved
RUN go get gopkg.in/DuoSoftware/DVP-ARDSLiteRoutingEngineImproved.$MAJOR_VER/ArdsLiteRoutingEngineImproved

# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
#RUN go install github.com/DuoSoftware/DVP-ARDSLiteRoutingEngineImproved/ArdsLiteRoutingEngineImproved
RUN go install gopkg.in/DuoSoftware/DVP-ARDSLiteRoutingEngineImproved.$MAJOR_VER/ArdsLiteRoutingEngineImproved

# Run the outyet command by default when the container starts.
ENTRYPOINT /go/bin/ArdsLiteRoutingEngineImproved

# Document that the service listens on port 8835.
EXPOSE 8835