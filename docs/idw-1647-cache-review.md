# IDW-1647 · Public video cache review

22 November 2026 · Local implementation and triage decision

**Keep production reads and browser writes available.** MarketViews has a published video integration that can need a browser write. Public creation and retrieval of a new entry are expected Prebid behavior. The report does not establish substitution of a winning creative, overwrite of a live UUID, or production resource exhaustion. The request handler did have an independent resource-control gap: it buffered an entire POST before enforcing item count and per-item size. This branch bounds that allocation per request. Aggregate traffic and backend capacity still need separate controls.

## Evidence behind the decision

| Source | Finding | Limit of the evidence |
| --- | --- | --- |
| [IDW-1652 placement review](https://dianomi.atlassian.net/browse/IDW-1652), [published /dianomi/](https://www.marketviews.com/dianomi/) and [/retirement-gam/](https://www.marketviews.com/retirement-gam/) | Both pages use the same MarketViews GAM unit and Dianomi cache. An eligible VAST bid without a reusable key can trigger a browser write. The two observed November auctions returned no bids. The separate /retirement/ integration points at an external cache. | Two empty auctions cannot measure how often winning bids need the fallback. The inline players on these pages are separate from the web SDK. |
| [GAM order 4090132229](https://admanager.google.com/76291182#delivery/order/order_overview/order_id=4090132229) and the [parent investigation](https://dianomi.atlassian.net/browse/IDW-1647) | The parent records 37 order impressions, five impressions and four completions for line item 7328134272, and seven instream unit impressions in the 20 October–18 November report. The owner described low-volume MarketViews testing. Four of 11 inspected VAST creatives point at the production cache. | Order, line item and ad unit measure different populations. These figures do not identify a cache writer or attribute a completion to a particular creative. |
| 25 February MarketViews browser capture, reviewed under IDW-1652 | Three distinct UUIDs sent to GAM were absent from the recorded PBS response. Returned GAM wrappers contained their respective break's UUID. | The capture omits the cache write, subsequent playback read and ad start. It cannot establish writer identity or November usage. |
| [IDW-1651 infrastructure review](https://dianomi.atlassian.net/browse/IDW-1651) | Infrastructure reported one day of ALB logs, no WAF and no POST identified in its US East sample. | The displayed top-20 results omit the method column. Neither all-region usage nor a zero-write conclusion follows. Current production AWS credentials were unavailable for an independent check. |

## Controls verified in the fork

At baseline `57bbb13`, the public write route is configurable. The private/admin handler always registers it. Checked-in settings allow public POST, ignore caller-supplied keys, limit a batch to 10 values, limit each stored value to 10 KiB before compression and configure a 3,600-second maximum TTL. The Redis backend uses an atomic conditional write with expiry. Existing keys survive a collision. Generated keys use random version-4 UUIDs. The public router accepts cross-origin requests and configures a rate limit of 100 requests per second using forwarded client IP headers. Trusted proxy handling, effective deployment values, backend selection, memory ceiling and eviction policy require confirmation in each region. The sample configuration selects memory, whose implementation does not expire entries.

Source references: [routes](../endpoints/routing/handler.go), [defaults](../config.yaml), [backend construction](../backends/config/config.go), [Redis writer](../backends/redis.go), [memory writer](../backends/memory.go) and [key generation](../utils/utils.go).

## Local remediation and rollout decision

The cache branch applies a configurable 1 MiB default to the full JSON body on both write routes. A request that exceeds the limit, including a streamed request, receives HTTP 413 before the first backend write. Invalid JSON receives a fixed error message. The separate SDK branch binds fallback VAST selection to the ad, break and auction. It preserves a supplied VAST UUID and keeps bidder-specific and generic targeting separate when they identify different ads.

Before deployment, compare legitimate request sizes and batch sizes with the proposed cap in every region. Canary with a representative winning video placement and compare HTTP 413 rates, cache write success, GAM impressions, ad starts and completions with baseline. If legitimate writes regress, restore the previous release or raise the cap to a measured compatible value. Verify recovery against the same signals.

## Evidence still needed for closure

- Infrastructure: identify the deployed cache build and effective settings per region. Confirm the public and private listeners, trusted source-IP handling and rate limits, TTL, Redis/Valkey memory limit and eviction policy. Correlate memory pressure, evictions, rejected writes and latency with traffic before assigning production impact.
- Infrastructure: report ALB counts by region, exact cache path, method, status and time window, including successful GET and POST. Distinguish browser from backend writers only where source evidence permits. Obtain CloudWatch target-group RequestCount for 7 and 30 days as a traffic cross-check. That metric aggregates methods. State missing retention periods and avoid extrapolating a daily sample to a month.
- Placement owner: capture one ordinary winning auction from originating bid through returned cache UUID, GAM creative, matching playback GET and ad start. Preserve the join using redacted identifiers. Inventory other publisher-managed integrations before retiring the public write route.
- Reassess poisoning only with evidence linking an independently created entry to another auction's legitimate playback, or evidence of overwrite. Document measured availability or cost impact separately. If all browser dependencies are retired or migrated, verify backend writes use the private route before a staged public-POST restriction. Keep public GET available and retain a tested route rollback.

## Draft response to the reporter

Thank you for documenting the public cache behavior. Our review confirms that public writes can support publisher video delivery. We have not established that creating an entry lets a caller replace another auction’s creative or overwrite its cache entry. We identified a separate gap in request-size handling and prepared a local change to bound the complete write request. Production traffic and capacity checks are still open. If you already have a redacted trace linking an unrelated cache entry to an actual winning auction and playback, it would help us assess the reported impact.
