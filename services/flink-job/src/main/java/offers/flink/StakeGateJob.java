package offers.flink;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.SimpleStringSchema;
import org.apache.flink.api.java.functions.KeySelector;
import org.apache.flink.api.java.tuple.Tuple2;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.windowing.ProcessWindowFunction;
import org.apache.flink.streaming.api.windowing.assigners.TumblingProcessingTimeWindows;
import org.apache.flink.streaming.api.windowing.time.Time;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;

import java.util.ArrayList;
import java.util.List;
import java.util.Properties;

import org.apache.kafka.clients.admin.AdminClient;
import org.apache.kafka.clients.admin.AdminClientConfig;
import org.apache.kafka.clients.admin.CreateTopicsResult;
import org.apache.kafka.clients.admin.DescribeTopicsResult;
import org.apache.kafka.clients.admin.NewTopic;
import org.apache.kafka.common.errors.TopicExistsException;

public class StakeGateJob {
    public static class BetPlaced {
        public String eventId;
        public long ts;
        public String userId;
        public String market;
        public double stake;
        public String currency;
    }

    static String getenv(String k, String def) {
        String v = System.getenv(k);
        return v == null || v.isEmpty() ? def : v;
    }

    public static void main(String[] args) throws Exception {
        final String brokers = getenv("KAFKA_BROKERS", "redpanda:9092");
        final String inputTopic = getenv("INPUT_TOPIC", "events.bet_placed");
        final String outputTopic = getenv("OUTPUT_TOPIC", "events.bet_qualified");
        final double threshold = Double.parseDouble(getenv("MIN_WINDOW_STAKE", "50"));

        final ObjectMapper mapper = new ObjectMapper()
                .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Ensure topics exist before starting the Kafka source to avoid metadata races
        ensureTopics(brokers, inputTopic, outputTopic);

        KafkaSource<String> source = KafkaSource.<String>builder()
                .setBootstrapServers(brokers)
                .setTopics(inputTopic)
                .setGroupId("flink-stake-gate")
                .setStartingOffsets(OffsetsInitializer.earliest())
                .setValueOnlyDeserializer(new SimpleStringSchema())
                .build();

        DataStream<String> raw = env.fromSource(source, WatermarkStrategy.noWatermarks(), "bets-source");
        DataStream<BetPlaced> bets = raw.map(s -> mapper.readValue(s, BetPlaced.class));

        DataStream<String> qualifiedJson = bets
                .keyBy((KeySelector<BetPlaced, String>) b -> b.userId)
                .window(TumblingProcessingTimeWindows.of(Time.minutes(2)))
                .process(new ProcessWindowFunction<BetPlaced, BetPlaced, String, TimeWindow>() {
                    @Override
                    public void process(String key, Context ctx, Iterable<BetPlaced> elements, Collector<BetPlaced> out) {
                        double sum = 0.0;
                        List<BetPlaced> buf = new ArrayList<>();
                        for (BetPlaced b : elements) { sum += b.stake; buf.add(b); }
                        if (sum >= threshold) {
                            for (BetPlaced b : buf) out.collect(b);
                        }
                    }
                })
                .map(b -> mapper.writeValueAsString(b));

        KafkaSink<String> sink = KafkaSink.<String>builder()
                .setBootstrapServers(brokers)
                .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                        .setTopic(outputTopic)
                        .setValueSerializationSchema(new SimpleStringSchema())
                        .build())
                .build();

        qualifiedJson.sinkTo(sink);
        env.execute("Stake Gate (2m tumbling, min=" + threshold + ")");
    }

    static void ensureTopics(String brokers, String... topics) throws Exception {
        Properties props = new Properties();
        props.put(AdminClientConfig.BOOTSTRAP_SERVERS_CONFIG, brokers);
        try (AdminClient admin = AdminClient.create(props)) {
            // Create topics if missing
            List<NewTopic> toCreate = new ArrayList<>();
            for (String t : topics) {
                toCreate.add(new NewTopic(t, 1, (short) 1));
            }
            try {
                CreateTopicsResult res = admin.createTopics(toCreate);
                // ignore TopicExistsException
                res.all().whenComplete((v, ex) -> { /* no-op */ });
            } catch (TopicExistsException ignored) {}
            // Block until metadata is available
            DescribeTopicsResult dtr = admin.describeTopics(java.util.Arrays.asList(topics));
            dtr.all().get();
        }
    }
}
