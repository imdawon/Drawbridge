# Emissary Client Data Model

An authenticated Emissary client needs to be able to be uniquely identified to ensure Drawbridge can allow Emissary
to only access the resources an admin configures.  

To Drawbridge, an Emissary client has the following features about it:
- A unique ID (uuid)
- Operating Systems Details (hostname, version)
- Timestamp of last successfully evaluated Emissary config policy.  