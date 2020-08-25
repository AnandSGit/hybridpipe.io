################################################################################################ 
# BUILD : docker build --tag dockqpid . (End "." would take the Dockerfile from the local folder, this file)
# RUN   : docker run -ti dockqpid
################################################################################################

FROM ubuntu:latest
MAINTAINER "dev@qpid.apache.org"
ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
	apt-get install -y \
	gcc \
	g++ \
	automake \
	libwebsockets-dev \
	libtool \
	zlib1g-dev \
	cmake \
	libsasl2-dev \
	libssl-dev \
	python \
	python-dev \
	libuv1-dev \
	sasl2-bin \
	swig \
	maven \
	git && \
	apt-get -y clean

RUN git clone https://gitbox.apache.org/repos/asf/qpid-dispatch.git && \
	cd /qpid-dispatch && \
	git submodule add https://gitbox.apache.org/repos/asf/qpid-proton.git && \
	git submodule update --init

WORKDIR /qpid-dispatch

RUN mkdir qpid-proton/build && \
	cd qpid-proton/build && \
	cmake .. -DSYSINSTALL_BINDINGS=ON -DCMAKE_INSTALL_PREFIX=/usr -DSYSINSTALL_PYTHON=ON && \
	make install

WORKDIR /qpid-dispatch

RUN mkdir build && cd build && \
	cmake .. -DCMAKE_INSTALL_PREFIX=/usr -DUSE_VALGRIND=NO && \
	cmake --build . --target install

ENV PYTHONPATH=/usr/lib/python2.7/site-packages
####<Not running for now>#### RUN cd /qpid-dispatch/build && ctest -VV
WORKDIR /qpid-dispatch

# Start the dispatch router
ENTRYPOINT ["qdrouterd"]
#### CMD ["/bin/bash"]
