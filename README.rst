Switch Port Ops
================

Online/Offline switch port with standard SNMP protocol:

- No switch model dependency;
- No vendor dependency.

Run the Service
----------------

Either run from CLI directly or as a docker container.

Run from CLI
~~~~~~~~~~~~~~

::

  go build .
  ./swportops

Run with Docker
~~~~~~~~~~~~~~~~~

::

  docker build -t swportops .
  docker run -d --rm --name swportops -p 8080:8080 swportops

Use the Service
-----------------

1. Get OIDs for switch ports:

   ::

     curl -G http://127.0.0.1:8080/ports -d 'switch=192.168.1.1' -d 'community=private'

2. Set port as online/offline using its OID:

   ::

     curl -G http://127.0.0.1:8080/port -d 'switch=192.168.1.1' -d 'community=private' -d 'oid=.1.3.6.1.2.1.2.2.1.7.101191680' -d 'ops=off'
     curl -G http://127.0.0.1:8080/port -d 'switch=192.168.1.1' -d 'community=private' -d 'oid=.1.3.6.1.2.1.2.2.1.7.101191680' -d 'ops=on'

Note
-----

- Accessing **/ports** will cost some time since a SNMP walk operation will be performed against the full interface MIB tree under ifName (.1.3.6.1.2.1.31.1.1.1.9). This can be smoothed if the service is running close to the switch location;
- The community string should have write permission since on/off a port needs such a permission;
