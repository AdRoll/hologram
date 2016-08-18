Hologram Server
===============

The Hologram Server manages AWS credentials for a team of developers, allocating temporary credentials using AWS STS to
developers on request. It is designed to work with the Hologram Agent, responding to authenticated requests for
credentials with fresh or cached credentials from AWS.


protobuf server
---------------

Hologram accepts TCP connections on port 3100, receiving and responding to messages using a Protocol Buffers-based format.


LDAP
----

Hologram supports a pluggable authentication and authorization mechanism, and the default implementation avialable is
LDAP. Users authenticate to Hologram using an SSH key challenge, and Hologram looks up the SSH public keys to use in
LDAP.

AWS client
----------

AWS STS is used to generate temporary credentials.

logging
-------

All authentications, whether successful or not, can be logged to Amazon SimpleDB to provide an audit trail.
