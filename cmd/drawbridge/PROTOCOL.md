# Drawbridge Protocol
The Drawbridge Protocol is a command syntax for mTLS-encrypted TCP tunnels - used for communications between Emissary clients and Drawbridge servers. 

Simply put: we're just writing strings over the wire that tell Drawbridge to do certain things. Drawbridge will only accept commands from Emissary clients with valid mTLS certificates.


## Drawbridge Protocol Details
The Drawbridge Protocol uses various premable commands with the following format:

<COMMAND_NAME> (7 chars) <SERVICE_ID> (3 chars)

### Drawbridge Protocol Commands
You may notice there are some prefixes to each command, such as `PS` and `OB`.

`PS` is shorthand for `Protected Service`, a service protected by Drawbridge as defined by the Drawbridge admin e.g a Minecraft server.

`OB` is shorthand for `Outbound Service`, when an Emissary client can act as a proxy for a locally-accessible networked service, such as an internal analytics dashboard. Emissary's Outbound feature cab exposes this local service to Drawbridge, so other Emissary clients can connect to it.

### Emissary Outbound Services: Explained
Emissary Outbound Services are a great way to expose a service to Drawbridge without needing to forward a port! You can use this to allow access to networked services from any network, as long as you have an existing Drawbridge server and a valid Emissary client.

For example, say I have a Drawbridge server set up on my home network. I allow my friend, Matt, to access this Drawbridge server via an Emissary Bundle I've sent to him.

Matt has a Plex movie server on his LAN that is not accessible outside of his network. He can create an Emissary Outbound connection to his local Plex server, which exposes it to our Drawbridge server. That way, any Emissary user on our Drawbridge server can now access Matt's Plex server without him needing to forward a port on his router!

## Drawbridge Protocol - Command Names Explained

  ### PS_LIST
  - A request from an Emissary client to get the list of Drawbridge Protected Services. 
  
  Format: comma-delimited list: 
  `<3 digit Protected Service identifier><service name as set by Drawbridge admin>,<additional services>`
    
  #### Emissary Request Example Value (Sent by Emissary client to Drawbridge)
  - PS_LIST
  
  #### Drawbridge Response Example Value (Sent by Drawbridge back to Emissary client)
  - PS_LIST: 001aaa,\n\n
    
  ### PS_CONN
  - A request from an Emissary Client to start proxying data to a Protected Service.

  Format: `<3 digit Protected Service identifier>`
      
    #### Emissary Request Example Value (Sent by Emissary client to Drawbridge)
    - PS_CONN
    **Note that this starts a proxy tunnel between Drawbridge and a Protected Service, proxying data as follows:**
    Emissary Client <-> Drawbridge <-> Protected Service      


  ## OB_CR8T
## Drawbridge Behavior Cycle

The foundation of Drawbridge's security model consists of the following:
- Deny-by-default: only allow connections to Drawbridge from Emissary clients with valid Drawbridge-generated mTLS certificates.
- Each mTLS certificate is scoped to a single service e.g. your mTLS certificate for a self-hosted http application, protected
  by Drawbridge, will not be valid for your Minecraft server. (not yet implemented)