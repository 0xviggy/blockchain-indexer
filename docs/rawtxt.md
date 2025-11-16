GO:

routine
context - propagation within and across 2 service
http - de/serialization
io package (reader writer stream processing)
channel
sync package (mutex, read write locks, wait group)
require package for unit testing

googles guide for idiomatic programming



Misc:
- JWT, Authentication cookie. What is identity provider. LDAP, SSO 
- CAP & PACELC theorems
- Why is it easy to scale stateless & so hard for scaling stateful services?
- Scaling the data layer (database & event streams)


Postgre SQL
- MVCC multi version concurrency control. Handling concurrent transactions
- Row updates.
- Isolation levels (Serializable & others. Read Commited etc). Study this with example code snippets.
- Transaction handling code in Golang. (Begining a transaction & correctly handling rollback on error condition. And no rollback if properly commited.) 
- Schema upgrade management tool. (Industry adoption tools). Apache avro, Liquibase**.
- Master master replication. Multi tiered slave replication.
- Replication is not a strategy to scale data set size or concurrent writes. It scales concurrent reads with an eventually consistent read model. You may read stale data till "some" time after it is written. 
- How to achieve Write scalability in SQL? (Partitioning maybe?).
- Hilite how partioning data can make computation of certain type of queries that spans multiple shards, can get very difficult.

Cover some NO SQL DBs:- (Draw pros cons comparisons bw these)
- Cassandra. (Self healing, session coordinator, qouroum consistency, write updates. Why deletes are expensive?)
- Dynamo (Conflict handling left to client, eventually consistent, )
- Mongo 

Event driven architecture
- Idempotent consumers. Are you OK with out of order event consumption? (Then you can use queue based configuration on a broker for consumption of events).
- Event broker like Kafka is a great option for in order event consumption.
- Event broker provides an append only ordered log of "facts" that occur in the various streams of the distributed system. It provides storage for events even after they are consumed. Event broker also provides configuration to broadcast an event to all instances of a consumer service.
- A message broker does not typically provide storage for the event after it has been consumed. Hilite the difference between a message broker & an event broker.
- AMQP (Advanced message queueing protocol 0.9.1). Little bit on rabbitmq if you have used it. Other cloud service providers for AMQP are AWS's SQS/SNS.
- How to scale the event stream?
- What is consumer lag in EDA? What implication its value has on the scalability needs of the consumer service.
- Schematization & upgrade management for the "event definition" in EDA? Usage of Tools like apache avro & Google protobuf?


