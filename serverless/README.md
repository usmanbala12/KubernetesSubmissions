I encountered a problem setting up Knative due to a version incompatibility between my k3d kubernetes version Knative v1.15.0. installing Knative v1.14.0 fixed the issue

kn service create hello\
--image gcr.io/knative-samples/helloworld-go \
--port 8080 \
--env TARGET=World

kn service update hello\
--env TARGET=Knative

 kn service update hello\
--traffic hello-00001=50 \
--traffic @latest=50
