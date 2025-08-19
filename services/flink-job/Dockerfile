FROM maven:3.9-eclipse-temurin-17 AS build
WORKDIR /app
COPY pom.xml .
RUN mvn -q -B dependency:go-offline
COPY src ./src
RUN mvn -q -B -DskipTests package

FROM flink:1.18.1-scala_2.12-java11
# The shaded jar is the main artifact (no "-shaded" classifier). Copy it in place.
COPY --from=build /app/target/flink-stake-gate-0.1.0.jar /opt/flink/usrlib/job.jar
ENV KAFKA_BROKERS=redpanda:9092 \
    INPUT_TOPIC=events.bet_placed \
    OUTPUT_TOPIC=events.bet_qualified \
    MIN_WINDOW_STAKE=50
CMD ["bash","-lc","/opt/flink/bin/flink run -m flink-jobmanager:8081 /opt/flink/usrlib/job.jar"]
