# Development notes

## Design decisions

- Jetstreams are created/deleted when the network is created/deleted.
- There is a special user that has access rights to create jetstreams for the account.
  This user is used by anck.
## Known limitations

- Due to the fact that anck-credentials currently does not provide a way to
  retrieve credentials per component, the same nats credentials (jwt, nkey)
  are shared between all components.
