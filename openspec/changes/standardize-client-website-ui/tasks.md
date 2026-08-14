## 1. Baseline and canonical standard

- [x] 1.1 Run the existing App test suite and website test/build commands before styling changes, and record any pre-existing failures separately from this change.
- [x] 1.2 Create root `DESIGN.md` with the approved scope, gold-led brand principles, semantic color table, light/client-dark themes, typography, spacing, radii, shadows and platform unit mappings from `design.md`.
- [x] 1.3 Add button, card, form, chip/tag, navigation, overlay, empty/loading/skeleton, responsive, accessibility and motion rules to `DESIGN.md`.
- [x] 1.4 Add a route/component visual QA matrix and a new-page review checklist to `DESIGN.md`, covering App light/dark, WeChat mini-program, and website mobile/desktop states.
- [x] 1.5 Reference root `DESIGN.md` from `app/AGENTS.md`, the synchronized `app/GEMINI.md`, and `website/AGENTS.md` without changing unrelated guidance.

## 2. App and mini-program token foundation

- [x] 2.1 Expand the light and `.dark-mode` CSS variables in `app/App.vue` to the documented `--ui-*` brand, surface, text, border, semantic, radius and shadow roles while preserving temporary compatibility aliases during migration.
- [x] 2.2 Align `app/theme.json` light/dark native background, text, border, tab and switch values with the documented warm-neutral token mapping.
- [x] 2.3 Refactor `app/utils/theme-application.js` so navigation bar, page background and TabBar styles match the native theme tokens for both themes without changing route behavior.
- [x] 2.4 Align `app/uni.scss` UniApp primary, success, warning, error, text, background, border, spacing and radius defaults with the public UI standard.
- [x] 2.5 Extend `app/tests/theme-platform-config.test.mjs` and `app/tests/theme-application.test.mjs` to verify the core token values remain synchronized across native and runtime theme entry points.
- [x] 2.6 Add `app/tests/ui-style-contract.test.mjs` to verify root `DESIGN.md`, foundational token availability, prohibited legacy cool surfaces and narrowly documented exceptions.

## 3. Website token foundation

- [x] 3.1 Define the shared `--ui-*` brand, surface, text, border, semantic, radius and shadow tokens in `website/assets/styles/main.css` and migrate the global body background and text defaults to them.
- [x] 3.2 Add global website focus-visible, pressed, disabled and reusable CTA/card primitives only where the same pattern is consumed by multiple components.
- [x] 3.3 Add `website/tests/ui-style-contract.test.ts` to verify required tokens exist and migrated shared components do not hardcode core brand or foundational neutral literals.

## 4. Website component and page conformance

- [x] 4.1 Migrate `website/components/SiteHeader.vue` and `SiteFooter.vue` to the shared tokens, including keyboard focus and narrow-screen navigation behavior.
- [x] 4.2 Migrate `website/components/HeroSection.vue` and `PageHero.vue` to the standard title hierarchy, brand actions, surfaces, radii and responsive spacing.
- [x] 4.3 Migrate `FeatureGrid.vue`, `ScenarioCards.vue` and `FaqList.vue` to standard section headings, cards, borders, text roles and mobile grids.
- [x] 4.4 Migrate `DownloadPanel.vue`, `website/pages/download.vue` and `website/pages/coming-soon.vue` to the shared CTA, platform card and status styles.
- [x] 4.5 Migrate `LegalArticle.vue`, `website/pages/privacy.vue`, `agreement.vue` and `contact.vue` to the standard reading surfaces, form/control states and text hierarchy.
- [x] 4.6 Audit every `website/pages/*.vue` and shared component for remaining duplicated core literals, inconsistent radii, horizontal overflow and missing focus-visible states, fixing only conformance issues.

## 5. App entry flow, main tabs and shared overlays

- [x] 5.1 Migrate `app/pages/welcome/index.scss` and `app/pages/login/login.vue` to the canonical entry-flow tokens while retaining green exclusively for WeChat login and semantic success.
- [x] 5.2 Migrate `app/components/agreementConsentSheet.vue`, `bindPhone.vue` and `gameTypeModal.vue` to standard modal surfaces, form controls, button states and bottom safe-area handling.
- [x] 5.3 Migrate `app/pages/index/index.scss` and its inline visual bindings to the shared page, hero, summary card, ranking preview, venue preview, empty and skeleton roles.
- [x] 5.4 Migrate `app/pages/match/index.scss` and its inline visual bindings to standard toolbar, filter sheet, match card, QR overlay, referee preview and state semantics.
- [x] 5.5 Migrate `app/pages/social/index.scss` and its inline visual bindings to standard page, filter, event card, status and empty/loading semantics.
- [x] 5.6 Migrate `app/pages/user/index.scss` and its inline visual bindings to standard profile, identity, status, metrics, shortcut, service, QR and reward-entry semantics.
- [x] 5.7 Migrate `app/pages/ranking/index.scss` to warm-neutral light and warm-black dark tokens, remove duplicate style blocks, and normalize mixed `px`/`rpx` usage where it affects the same component.

## 6. Match and user subpage conformance

- [x] 6.1 Migrate `app/subPages/match/matchDetail.scss` and page-local styles to standard detail cards, participant, score, metadata and dark surfaces.
- [x] 6.2 Migrate `app/subPages/match/opponentSelector.scss` and page-local styles to standard search, selection card, button and empty/loading states.
- [x] 6.3 Migrate `app/subPages/match/playing.scss` to warm-black dark surfaces and semantic scoring/status colors without changing real-time match behavior.
- [x] 6.4 Migrate `app/subPages/match/matchResult.scss` to canonical result, confirmation, action and dark-theme tokens without changing result flow.
- [x] 6.5 Migrate `app/subPages/match/shareResult.vue` to standard page surfaces and action controls while preserving its poster-specific dark canvas.
- [x] 6.6 Migrate `matchHistory`, `h2hRecord` and `opponentRecord` user subpages to standard list, filter, card, score and empty/loading states.
- [x] 6.7 Migrate `rankExplain`, `reputation` and `statsDetail` user subpages to standard reading surfaces, semantic progress/status colors and warm dark theme.
- [x] 6.8 Migrate `memberCenter`, `editProfile` and `settings` user subpages to standard premium, form, list-row, button and disabled states.
- [x] 6.9 Migrate `app/subPages/user/notification.vue` and `notification.scss` to standard preference rows, switches, dividers and warm dark surfaces.

## 7. Social, tournament and venue conformance

- [x] 7.1 Migrate social `feed.vue`, `postCreate.vue` and `myPosts.vue` to standard composer, post card, action, form and empty/loading states.
- [x] 7.2 Migrate `friendList.vue`, `addFriend.vue`, `friendRequests.vue` and `friendHomepage.scss` to standard profile/list/search/request states and warm dark surfaces.
- [x] 7.3 Migrate `challenges.vue` and `pkReport.scss` to standard competitive cards, score emphasis, filters, modal and semantic result colors.
- [x] 7.4 Migrate tournament `index.scss` and `create.scss` away from legacy cool foundational gradients while preserving game/status information colors.
- [x] 7.5 Migrate tournament `detail.scss` and `bracket.vue` to standard event surfaces, metadata, bracket cards and responsive/scroll behavior.
- [x] 7.6 Migrate venue `index.vue` and `detail.scss` to standard location header, venue cards, detail sections, actions and empty/loading states.
- [x] 7.7 Migrate `app/subPages/venue/submit.scss` to standard form, helper, validation, submit action and warm-neutral page surfaces.

## 8. Remaining registered page conformance

- [x] 8.1 Migrate `app/subPages/season/index.vue` and `report.vue` to standard season cards, report surfaces, chart containers and warm-black dark theme.
- [x] 8.2 Migrate achievement `index.scss` and `detail.vue` to standard honor, progress, locked/unlocked, filter and detail semantics.
- [x] 8.3 Migrate rules `index.vue`, `detail.vue` and `glossary.vue` to standard reading, search, section, term and empty states.
- [x] 8.4 Audit the App agreement pages so their typography, links, cards and themes use the shared tokens without changing legal content.
- [x] 8.5 Migrate `app/subPages/help/feedback.scss` to warm-neutral form and warm-black dark surfaces while preserving complaint/report validation behavior.
- [x] 8.6 Migrate `app/subPages/notification/index.vue` to standard notification cards, unread state, filters, empty/loading states and dark surfaces.

## 9. Cross-cutting conformance audit

- [x] 9.1 Audit every custom App `button` touched by this change for `margin: 0`, hidden `button::after`, matching height/line-height, white primary text and `[disabled]` styling.
- [x] 9.2 Audit all touched client overlays and bottom sheets for tab bar coordination, `env(safe-area-inset-bottom)` spacing and non-overlapping actions.
- [x] 9.3 Audit all registered App pages for consistent empty, loading, error and skeleton hierarchy, removing only duplicate styles made obsolete by this migration.
- [x] 9.4 Run a repository style scan to confirm core brand/foundational literals are centralized, prohibited cool page/card/input/modal surfaces are gone, and every exception is documented in the contract-test allowlist.
- [x] 9.5 Confirm the migration did not change routes, API calls, lifecycle hooks, WebSocket behavior, form validation or content data.

## 10. Verification and completion

- [x] 10.1 Run `cd app && node --test tests/*.test.mjs` and fix all regressions introduced by the UI standardization.
- [x] 10.2 Run `cd website && npm run test` and fix all regressions introduced by token and component migration.
- [x] 10.3 Run `cd website && npm run build` and resolve any styling-related build or Sass warnings introduced by this change.
- [ ] 10.4 Manually verify the welcome/login/home/main-tab and representative list/detail/form/modal flows in App light and dark themes.
- [ ] 10.5 Manually verify the same representative client flows in WeChat mini-program, including native buttons, navigation bars, safe areas and bottom sheets.
- [ ] 10.6 Manually verify all website routes at mobile and desktop widths, including keyboard focus, CTA reachability, card reflow and absence of horizontal overflow.
- [x] 10.7 Remove only imports, local variables and duplicate rules made unused by this change, then confirm the final diff remains limited to public UI standards and conformance.
