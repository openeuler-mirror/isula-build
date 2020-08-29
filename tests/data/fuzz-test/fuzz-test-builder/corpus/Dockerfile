
LABEL os.type="linux"
LABEL platform.type="x86"

RUN mkdir /myvol
RUN echo "hello world" > /myvol/greeting

VOLUME /myvol

WORKDIR /a
WORKDIR b
WORKDIR c

ENV xxx=yyy

RUN echo $xxx

CMD [ "sleep", "999" ]
