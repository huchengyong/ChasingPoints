## ADDED Requirements

### Requirement: Canonical public UI standard
The repository SHALL provide a root-level `DESIGN.md` as the canonical visual standard for `app/` and `website/`, and the document MUST explicitly exclude `admin/` unless a later change expands its scope.

#### Scenario: Maintainer looks for UI guidance
- **WHEN** a maintainer prepares a new or changed page in `app/` or `website/`
- **THEN** the repository provides one discoverable `DESIGN.md` covering the applicable brand, token, component, responsive, theme and accessibility rules

#### Scenario: Admin UI is changed independently
- **WHEN** a maintainer works only in `admin/`
- **THEN** the public UI standard does not require the admin interface to adopt the website and client visual system

### Requirement: Shared semantic token contract
The public interfaces SHALL use a shared semantic token vocabulary for brand colors, surfaces, text, borders, status colors, spacing, radii and shadows, and each runtime SHALL map those semantics to documented platform-appropriate units and implementation files.

#### Scenario: Brand color is updated
- **WHEN** a core public brand token is changed
- **THEN** both the UniApp and Nuxt mappings expose the updated semantic role without requiring page authors to search and replace the color across individual pages

#### Scenario: Platform units differ
- **WHEN** the same spacing or radius role is rendered on Web and UniApp
- **THEN** each platform uses its documented local unit mapping while preserving the same component hierarchy and visual role

### Requirement: Gold brand and semantic status separation
The public interfaces MUST use gold for ordinary primary actions, selected states and brand emphasis, and MUST reserve green for WeChat login, success, online, in-progress or positive-change semantics. Warning, danger and informational states SHALL use their documented semantic colors rather than brand gold.

#### Scenario: Primary non-semantic action
- **WHEN** a page renders its ordinary primary action
- **THEN** the action uses the brand primary or approved brand gradient and white action text

#### Scenario: Success or live status
- **WHEN** a page communicates success, online availability, an in-progress state or positive change
- **THEN** the indicator uses the success semantic instead of treating gold as a success color

#### Scenario: Destructive action
- **WHEN** a user is offered a destructive or irreversible action
- **THEN** the action uses the danger semantic and is not styled as a normal gold primary action

### Requirement: App and mini-program theme conformance
Every route registered in `app/pages.json` SHALL continue to use the shared page theme mechanism and SHALL render documented page, card, text, border and control semantics in both light and dark client themes. Cool blue-black legacy colors MUST NOT serve as default page, card, input or modal surfaces after migration.

#### Scenario: Client page renders in light mode
- **WHEN** any registered App or mini-program page is opened with the light theme
- **THEN** its foundational surfaces and text use the documented warm-neutral light tokens

#### Scenario: Client page renders in dark mode
- **WHEN** any registered App or mini-program page is opened with the dark theme
- **THEN** its foundational surfaces and text use the documented warm-black and warm-neutral dark tokens

#### Scenario: Media overlay needs a cool dark color
- **WHEN** a media overlay, chart or other documented exception requires a non-foundational cool color
- **THEN** the exception is narrowly scoped and does not redefine the page, card, input or modal surface system

### Requirement: Website visual conformance
The website SHALL source its shared brand colors, surfaces, text colors, borders, radii and shadows from the website token definitions, and its shared components SHALL present the same gold-and-warm-neutral brand language as the client.

#### Scenario: User moves from website to client entry flow
- **WHEN** a user follows the website download journey and then opens the client welcome or login flow
- **THEN** both experiences use compatible brand emphasis, warm-neutral surfaces, rounded controls and typography hierarchy

#### Scenario: Shared website component is restyled
- **WHEN** Header, Hero, feature cards, scenario cards, download CTA, page hero, legal content or Footer consumes a core visual role
- **THEN** it references the website semantic token rather than duplicating the core brand literal in the component

### Requirement: Standard component states
Buttons, cards, form controls, chips, tags, navigation items, overlays, empty states, loading states and skeletons SHALL implement the documented default, active, disabled, error and dark-theme states applicable to their platform.

#### Scenario: Custom UniApp button is rendered
- **WHEN** a page renders a custom `button`
- **THEN** it removes the native after-border, resets native margin, vertically centers text with a matching height and line-height, and uses an attribute selector for its disabled state

#### Scenario: Empty or loading state is rendered
- **WHEN** a page has no content or is waiting for content
- **THEN** it uses the documented text hierarchy, spacing, surface and action semantics rather than introducing a page-specific neutral palette

#### Scenario: Modal covers the client bottom area
- **WHEN** a client overlay or bottom sheet covers the tab bar or device bottom edge
- **THEN** it preserves the safe area and prevents its actions from overlapping the tab bar

### Requirement: Responsive and accessible public UI
The public interfaces SHALL preserve readable contrast, visible focus or pressed feedback, non-color-only status communication and platform-appropriate target sizes. Website layouts SHALL remain usable on mobile and desktop viewports, and App/mini-program controls SHALL respect safe areas and native navigation constraints.

#### Scenario: Website is viewed on a narrow screen
- **WHEN** the website viewport reaches its documented mobile breakpoint
- **THEN** navigation, grids, cards and CTA groups reflow without horizontal overflow or inaccessible actions

#### Scenario: Client action is disabled
- **WHEN** a client action cannot be used
- **THEN** its disabled state remains legible, cannot be mistaken for an active action and is not communicated by color alone where status text is needed

#### Scenario: User navigates website by keyboard
- **WHEN** a keyboard user focuses an interactive website element
- **THEN** the element exposes a visible focus state with sufficient contrast

### Requirement: UI conformance guardrails
The repository SHALL provide automated checks for objective UI contracts and SHALL retain a documented manual visual QA matrix for aspects that cannot be reliably evaluated by static tests.

#### Scenario: Legacy foundational color is reintroduced
- **WHEN** a migrated client page assigns a prohibited legacy cool blue-black value to a page, card, input or modal background
- **THEN** the UI contract check fails unless the use is an explicitly documented exception

#### Scenario: Website component hardcodes a core token
- **WHEN** a migrated shared website component reintroduces a core brand or foundational neutral literal outside the token definition
- **THEN** the website UI contract check fails

#### Scenario: Change passes automated checks
- **WHEN** all UI contract tests pass
- **THEN** the implementation still requires manual review of representative App light/dark screens, WeChat mini-program screens, and website mobile/desktop layouts before completion
