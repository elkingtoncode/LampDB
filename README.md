![logo](/resource/doc/lamp_db.png?raw=true "lamp Labs logo")


[![Circle CI](https://circleci.com/gh/lampdb/lamp.svg?style=svg)](https://circleci.com/gh/lampdb/lamp) [![GoDoc](https://godoc.org/github.com/lampdb/lamp?status.png)](https://godoc.org/github.com/lampdb/lamp) ![Project Status](http://img.shields.io/badge/status-alpha-red.svg)

## A Scalable, Geo-Replicated, Transactional Datastore

**Table of Contents**

- [Status](#status)
- [Running lamp Locally](#running-lamp-locally)
- [Deploying lamp in production](#deploying-lamp-in-production)
- [Getting in touch and contributing](#get-in-touch)
- [Talks](#talks)
- [Design](#design) and [Datastore Goal Articulation](#datastore-goal-articulation)
- [Architecture](#architecture) and [Client Architecture](#client-architecture)

[![WIRED on lampdb](/resource/doc/wired-preview.png?raw=true)](http://www.wired.com/2014/07/lampdb/)

## Status

See our
[Roadmap](https://github.com/lampdb/lamp/wiki/Roadmap) and
[Issues](https://github.com/lampdb/lamp/issues)

## Running lamp Locally

Getting started is most convenient using a recent version (>1.2) of [Docker](http://docs.docker.com/installation/).

If you don't want to use Docker,
* set up the dev environment (see [CONTRIBUTING.md](CONTRIBUTING.md))
* `make build`
* ignore the initial calls to `docker` below.

#### Bootstrap and talk to a single node

Getting a lamp node up and running is easy. If you have the `lamp` binary, skip over the next shell session. Most users however will want to run the following:

```bash
# Get the latest image from the registry. Skip if you already have an image
# or if you built it yourself.
docker pull lampdb/lamp
# Open a shell on a lamp container.
docker run -t -i -p 8080:8080 lampdb/lamp shell
# root@82cb657cdc42:/lamp#
```

If the docker command fails but Docker is installed, you probably need to initialize it. Here's a common error message:
```bash
FATA[0000] Post http:///var/run/docker.sock/v1.17/images/create?fromImage=lampdb%2Flamp%3Alatest: dial unix /var/run/docker.sock: no such file or directory
```
On OSX:
```bash
# Setup Boot2Docker. This should only need to be done once.
boot2docker init
# Start Boot2Docker. This will need to be run once per reboot.
boot2docker start
# Setup environment variables. This will need to be run once per shell.
$(boot2docker shellinit)
```
Other operating systems will have a similar set of commands. Please check Docker's documentation for more info.


Now we're in an environment that has everything set up, and we start by first initializing the cluster and then firing up the node:

```bash
DIR=$(mktemp -d /tmp/dbXXXXXX)
# Initialize CA, server, and client certificates. Default directory is --certs=certs
./lamp cert create-ca
./lamp cert create-node 127.0.0.1 localhost $(hostname)
./lamp cert create-client root
# Initialize data directories.
./lamp init --stores=ssd=$DIR
# Start the server.
./lamp start --stores=ssd="$DIR" --gossip=self= &
```
This initializes and starts a single-node cluster in the background.

##### Built-in client

Now let's talk to this node. The easiest way to do that is to use the `lamp` binary - it comes with a simple built-in client:

```bash
# Put the values a->1, b->2, c->3, d->4.
./lamp kv put a 1 b 2 c 3 d 4
./lamp kv scan
# "a"     1
# "b"     2
# "c"     3
# "d"     4
# Scans do not include the end key.
./lamp kv scan b d
# "b"     2
# "c"     3
./lamp kv del c
./lamp kv scan
# "a"     1
# "b"     2
# "d"     4
# Counters are also available:
./lamp kv inc mycnt 5
# 5
./lamp kv inc mycnt -- -3
#2
./lamp kv get mycnt
#"\x00\x00\x00\x00\x00\x00\x00\x02"
```

Check out `./lamp help` to see all available commands.

#### Building the Docker images yourself
See [build/README.md](build/) for more information on the available Docker
images `lampdb/lamp` and `lampdb/lamp-dev`.
You can build both of these images yourself:

* `lampdb/lamp-dev`: `(cd build ; ./build-docker-dev.sh)`
* `lampdb/lamp`: `(cd build ; ./build-docker-deploy.sh)`
  (this will build the first image as well)

Once you've built your image, you may want to run the tests:
* `docker run "lampdb/lamp-dev" test`
* `make acceptance`

Assuming you've built `lampdb/lamp`, let's run a simple lamp node:

```bash
docker run -v /data -v /certs lampdb/lamp init --stores=ssd=/data
docker run --volumes-from=$(docker ps -q -n 1) lampdb/lamp \
  cert create-ca --certs=/certs
docker run --volumes-from=$(docker ps -q -n 1) lampdb/lamp \
  cert create-node --certs=/certs 127.0.0.1 localhost lampnode
docker run --volumes-from=$(docker ps -q -n 1) lampdb/lamp \
  cert create-client root
docker run -p 8080:8080 -h lampnode --volumes-from=$(docker ps -q -n 1) \
  lampdb/lamp start --certs=/certs --stores=ssd=/data --gossip=self=
```

Run `docker run lampdb/lamp help` to get an overview over the available commands and settings, and see [Running lamp](#running-lamp) for first steps on interacting with your new node.


## Deploying lamp in production

To run a lamp cluster on various cloud platforms using [docker machine](http://docs.docker.com/machine/),
see [lamp-prod](https://github.com/lampdb/lamp-prod)


## Get in touch

We spend almost all of our time here on GitHub, and use the [issue
tracker](https://github.com/lampdb/lamp/issues) for
bug reports and development-related questions.

For anything else, message our mailing list at [lamp-db@googlegroups.com](https://groups.google.com/forum/#!forum/lamp-db). We recommend joining before posting, or your messages may be held back for moderation.

### Contributing

We're an Open Source project and welcome contributions.
See [CONTRIBUTING.md](https://github.com/lampdb/lamp/blob/master/CONTRIBUTING.md) to get your local environment set up.
Once that's done, take a look at our [open issues](https://github.com/lampdb/lamp/issues/), in particular those with the [helpwanted label](https://github.com/lampdb/lamp/labels/helpwanted), and follow our [code reviews](https://github.com/lampdb/lamp/pulls/) to learn about our style and conventions.

## Talks

* [Venue: Data Driven NYC](https://youtu.be/TA-Jw78Ms_4), by [Spencer Kimball] (https://github.com/spencerkimball) on (06/16/2015), 23min.<br />
  A short, less technical presentation of lamp.
* [Venue: NY Enterprise Technology Meetup](https://www.youtube.com/watch?v=SXAEZlpsHNE), by [Tobias Schottdorf](https://github.com/tschottdorf) on (06/10/2015), 15min.<br />
  A short, non-technical talk with a small cluster survivability demo.
* [Venue: CoreOS Fest](https://www.youtube.com/watch?v=LI7uaaYeYmQ), by [Spencer Kimball](https://github.com/spencerkimball) on (05/27/2015), 25min.<br />
  An introduction to the goals and design of lamp DB. The recommended talk to watch if all you have time for is one.
* [Venue: The Go Devroom FOSDEM 2015](https://www.youtube.com/watch?v=ndKj77VW2eM&index=2&list=PLtLJO5JKE5YDK74RZm67xfwaDgeCj7oqb), by [Tobias Schottdorf](https://github.com/tschottdorf) on (03/04/2015), 45min.<br />
  The most technical talk given thus far, going through the implementation of transactions in some detail.

### Older talks

* [Venue: The NoSQL User Group Cologne](https://www.youtube.com/watch?v=jI3LiKhqN0E), by [Tobias Schottdorf](https://github.com/tschottdorf) on (11/5/2014), 1h25min.
* [Venue: Yelp!](http://www.youtube.com/watch?v=MEAuFgsmND0&feature=youtu.be), by [Spencer Kimball](https://github.com/spencerkimball) on (9/5/2014), 1h.


## Design

This is an overview. For an in depth discussion of the design, see the [design doc](https://github.com/lampdb/lamp/blob/master/docs/design.md).

For a quick design overview, see the [lamp tech talk slides](https://docs.google.com/presentation/d/1e3TOxImRg6_nyMZspXvzb2u43D6gnS5422vAIN7J1n8/edit?usp=sharing)
or watch a [presentation](#talks).


lamp is a distributed key/value datastore which supports ACID
transactional semantics and versioned values as first-class
features. The primary design goal is global consistency and
survivability, hence the name. lamp aims to tolerate disk,
machine, rack, and even datacenter failures with minimal latency
disruption and no manual intervention. lamp nodes are symmetric;
a design goal is one binary with minimal configuration and no required
auxiliary services.

lamp implements a single, monolithic sorted map from key to value
where both keys and values are byte strings (not unicode). lamp
scales linearly (theoretically up to 4 exabytes (4E) of logical
data). The map is composed of one or more ranges and each range is
backed by data stored in [RocksDB][0] (a variant of [LevelDB][1]), and is
replicated to a total of three or more lamp servers. Ranges are
defined by start and end keys. Ranges are merged and split to maintain
total byte size within a globally configurable min/max size
interval. Range sizes default to target 64M in order to facilitate
quick splits and merges and to distribute load at hotspots within a
key range. Range replicas are intended to be located in disparate
datacenters for survivability (e.g. { US-East, US-West, Japan }, {
Ireland, US-East, US-West}, { Ireland, US-East, US-West, Japan,
Australia }).

Single mutations to ranges are mediated via an instance of a
distributed consensus algorithm to ensure consistency. We’ve chosen to
use the [Raft consensus algorithm][2]. All consensus state is stored in
[RocksDB][0].

A single logical mutation may affect multiple key/value pairs. Logical
mutations have ACID transactional semantics. If all keys affected by a
logical mutation fall within the same range, atomicity and consistency
are guaranteed by [Raft][2]; this is the fast commit path. Otherwise, a
non-locking distributed commit protocol is employed between affected
ranges.

lamp provides snapshot isolation (SI) and serializable snapshot
isolation (SSI) semantics, allowing externally consistent, lock-free
reads and writes--both from an historical snapshot timestamp and from
the current wall clock time. SI provides lock-free reads and writes
but still allows write skew. SSI eliminates write skew, but introduces
a performance hit in the case of a contentious system. SSI is the
default isolation; clients must consciously decide to trade
correctness for performance. lamp implements a limited form of
linearalizability, providing ordering for any observer or chain of
observers.

Similar to [Spanner][3] directories, lamp allows configuration of
arbitrary zones of data. This allows replication factor, storage
device type, and/or datacenter location to be chosen to optimize
performance and/or availability. Unlike Spanner, zones are monolithic
and don’t allow movement of fine grained data on the level of entity
groups.

A [Megastore][4]-like message queue mechanism is also provided to 1)
efficiently sideline updates which can tolerate asynchronous execution
and 2) provide an integrated message queuing system for asynchronous
communication between distributed system components.

#### SQL - NoSQL - NewSQL Capabilities

![SQL - NoSQL - NewSQL Capabilities](/resource/doc/sql-nosql-newsql.png?raw=true)

## Datastore Goal Articulation

There are other important axes involved in data-stores which are less
well understood and/or explained. There is lots of cross-dependency,
but it's safe to segregate two more of them as (a) scan efficiency,
and (b) read vs write optimization.

#### Datastore Scan Efficiency Spectrum

Scan efficiency refers to the number of IO ops required to scan a set
of sorted adjacent rows matching a criteria. However, it's a
complicated topic, because of the options (or lack of options) for
controlling physical order in different systems.

* Some designs either default to or only support "heap organized"
  physical records (Oracle, MySQL, Postgres, SQLite, MongoDB). In this
  design, a naive sorted-scan of an index involves one IO op per
  record.
* In these systems it's possible to "fully cover" a sorted-query in an
  index with some write-amplification.
* In some systems it's possible to put the primary record data in a
  sorted btree instead of a heap-table (default in MySQL/Innodb,
  option in Oracle).
* Sorted-order LSM NoSQL could be considered index-organized-tables,
  with efficient scans by the row-key. (HBase).
* Some NoSQL is not optimized for sorted-order retrieval, because of
  hash-bucketing, primarily based on the Dynamo design. (Cassandra,
  Riak)

![Datastore Scan Efficiency Spectrum](/resource/doc/scan-efficiency.png?raw=true)

#### Read vs. Write Optimization Spectrum

Read vs write optimization is a product of the underlying sorted-order
data-structure used. Btrees are read-optimized. Hybrid write-deferred
trees are a balance of read-and-write optimizations (shuttle-trees,
fractal-trees, stratified-trees). LSM separates write-incorporation
into a separate step, offering a tunable amount of read-to-write
optimization. An "ideal" LSM at 0%-write-incorporation is a log, and
at 100%-write-incorporation is a btree.

The topic of LSM is confused by the fact that LSM is not an algorithm,
but a design pattern, and usage of LSM is hindered by the lack of a
de-facto optimal LSM design. LevelDB/RocksDB is one of the more
practical LSM implementations, but it is far from optimal. Popular
text-indicies like Lucene are non-general purpose instances of
write-optimized LSM.

Further, there is a dependency between access pattern
(read-modify-write vs blind-write and write-fraction), cache-hitrate,
and ideal sorted-order algorithm selection. At a certain
write-fraction and read-cache-hitrate, systems achieve higher total
throughput with write-optimized designs, at the cost of increased
worst-case read latency. As either write-fraction or
read-cache-hitrate applampes 1.0, write-optimized designs provide
dramatically better sustained system throughput when record-sizes are
small relative to IO sizes.

Given this information, data-stores can be sliced by their
sorted-order storage algorithm selection. Btree stores are
read-optimized (Oracle, SQLServer, Postgres, SQLite2, MySQL, MongoDB,
CouchDB), hybrid stores are read-optimized with better
write-throughput (Tokutek MySQL/MongoDB), while LSM-variants are
write-optimized (HBase, Cassandra, SQLite3/LSM, lamp).

![Read vs. Write Optimization Spectrum](/resource/doc/read-vs-write.png?raw=true)

## Architecture

lamp implements a layered architecture, with various
subdirectories implementing layers as appropriate. The highest level of
abstraction is the SQL layer (currently not implemented). It depends
directly on the [structured data API][5] ([structured/][6]). The structured
data API provides familiar relational concepts such as schemas,
tables, columns, and indexes. The structured data API in turn depends
on the [distributed key value store][7] ([kv/][8]). The distributed key
value store handles the details of range addressing to provide the
abstraction of a single, monolithic key value store. It communicates
with any number of [lamp nodes][9] ([server/][10]), storing the actual
data. Each node contains one or more [stores][11] ([storage/][12]), one per
physical device.

![lamp Architecture](/resource/doc/architecture.png?raw=true)

Each store contains potentially many ranges, the lowest-level unit of
key-value data. Ranges are replicated using the [Raft][2] consensus
protocol. The diagram below is a blown up version of stores from four
of the five nodes in the previous diagram. Each range is replicated
three ways using raft. The color coding shows associated range
replicas.

![Range Architecture Blowup](/resource/doc/architecture-blowup.png?raw=true)

## Client Architecture

lamp nodes serve client traffic using a fully-featured key/value
DB API which accepts requests as either application/x-protobuf or
application/json. Client implementations consist of an HTTP sender
(transport) and a transactional sender which implements a simple
exponential backoff / retry protocol, depending on lamp error
codes.

The DB client gateway accepts incoming requests and sends them
through a transaction coordinator, which handles transaction
heartbeats on behalf of clients, provides optimization pathways, and
resolves write intents on transaction commit or abort. The transaction
coordinator passes requests onto a distributed sender, which looks up
index metadata, caches the results, and routes internode RPC traffic
based on where the index metadata indicates keys are located in the
distributed cluster.

In addition to the gateway for external DB client traffic, each lamp
node provides the full key/value API (including all internal methods) via
a Go RPC server endpoint. The RPC server endpoint forwards requests to one
or more local stores depending on the specified key range.

Internally, each lamp node uses the Go implementation of the
lamp client in order to transactionally update system key/value
data; for example during split and merge operations to update index
metadata records. Unlike an external application, the internal client
eschews the HTTP sender and instead directly shares the transaction
coordinator and distributed sender used by the DB client gateway.

![Client Architecture](/resource/doc/client-architecture.png?raw=true)

[0]: http://rocksdb.org/
[1]: https://code.google.com/p/leveldb/
[2]: https://ramcloud.stanford.edu/wiki/download/attachments/11370504/raft.pdf
[3]: http://research.google.com/archive/spanner.html
[4]: http://research.google.com/pubs/pub36971.html
[5]: http://godoc.org/github.com/lampdb/lamp/structured
[6]: https://github.com/lampdb/lamp/tree/master/structured
[7]: http://godoc.org/github.com/lampdb/lamp/kv
[8]: https://github.com/lampdb/lamp/tree/master/kv
[9]: http://godoc.org/github.com/lampdb/lamp/server
[10]: https://github.com/lampdb/lamp/tree/master/server
[11]: http://godoc.org/github.com/lampdb/lamp/storage
[12]: https://github.com/lampdb/lamp/tree/master/storage
