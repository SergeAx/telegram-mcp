# Multi-account support: design notes

telegram-mcp serves a single account. These notes keep the decisions made
while designing its config storage, so that adding accounts later does not
reopen them.

## Already in place

- Each session is a file named after the account's Telegram user ID,
  `<user id>.session.age`, next to `config.toml`. The ID never changes and
  cannot clash, and logging in to the same account again replaces its file
  instead of adding another.
- `config.toml` names the account the server uses in `default_account`.
- One API ID and hash serve every account: Telegram gives a phone number
  one API ID, and one app can log in any number of accounts. There is no
  per-account `api_id`, as nothing shows that a session needs the API ID
  it was created with. Importing a Telegram Desktop session, which uses
  Desktop's own API ID, would be the case for an override.
- One age key encrypts every file.
- `auth` logs in to memory and names the session file once Telegram has
  said whose it is.

## One server for all accounts

- Every tool takes an `account` argument, a label or a user ID.
  - `tg_send` and `tg_read` require it, so that nothing is written to an
    account by default.
  - The read-only tools fall back to `default_account`, and `tg_dialogs`
    without `account` lists the dialogs of all accounts.
- Every response names the account it came from. Chat names and the access
  hashes behind them mean something only within one account, so the
  assistant has to send the account and the name together.
- `tg_me` lists every account: ID, label, name and username.
- Tool descriptions, built at startup, list the accounts. MCP's
  `instructions` would be the better place, but `metoro-io/mcp-golang`
  gives no way to set them.

A server per account was considered and rejected: it repeats every tool in
the assistant's context for each account, and cannot answer questions
across accounts.

## Accounts and labels

The accounts are the session files in the config directory. `config.toml`
only adds labels, set with `auth --label <label>`:

```toml
default_account = 3017676

[accounts.3017676]
label = "work"
```

Labels are needed because two accounts of one person usually share the
same name, and one may have no username.

## Per-account state

- Downloads go to `<download-dir>/<account>/<dialog>/`: chat names repeat
  across accounts.
- `tg.Client` holds one session store per account, so that concurrent tool
  calls share it.
- A per-account cache of chats and access hashes, which full Telegram
  clients keep next to each session, would belong next to its session file
  as well.

## Several processes

Every MCP client starts its own server, so several processes may write the
same session file. Atomic writes keep each file whole; there is no lock
across processes. gotd loads, changes and saves a session in separate
calls, so a lock would have to span the whole tool call and make the calls
to an account wait for each other. When two saves race, the loser's salt
or cached DC keys are lost, and gotd fetches them again. Re-running `auth`
while a tool call is in flight may save the previous login back over the
new one; that login is still valid, and another `auth` fixes it.
