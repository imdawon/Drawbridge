# Drawbridge Behavior Cycle

Drawbridge runs a tight ship, and is constantly evaluating the data it ingests from Emissary clients, as well as
running tasks on a schedule.

The foundation of Drawbridge's security model consists of the following:
- Deny-by-default by only allowing connections to Drawbridge from clients with valid mTLS certificates specifically for Drawbridge
- Each mTLS certificate is scoped to a single service e.g. your mTLS certificate for a self-hosted http application, protected
  by Drawbridge, will not be valid for your Minecraft server.