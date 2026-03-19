# Pitfalls Research: ACGWarehouse (二次元图片库)

**Domain:** Anime/Manga Image Gallery Management
**Researched:** 2026-03-14
**Confidence:** HIGH (based on 2024-2026 sources)

---

## Critical Pitfalls

### Pitfall 1: Storing Images as BLOBs in Database

**What goes wrong:**
Database performance degrades catastrophically as image collection grows. SQLite/PostgreSQL becomes slow for metadata queries because BLOBs bloat table pages and reduce cache efficiency.

**Why it happens:**
Developers treat images like any other data type. It is just a file leads to storing binary data in the same table as metadata.

**How to avoid:**
- Store images in filesystem or object storage (local disk for v1)
- Database only stores: file path, hash, dimensions, format
- Use content-addressed storage: images/{hash[:2]}/{hash[2:4]}/{hash}.ext
- Keep thumbnails in separate directory structure

**Warning signs:**
- Database file grows >1GB with <10k images
- Simple SELECT queries taking >100ms
- Backup time increases exponentially

**Phase to address:** Phase 1 (Image Scan and Storage Foundation)

---



### Pitfall 2: Perceptual Hashing False Positives/Negatives



**What goes wrong:**

Duplicate detection either misses actual duplicates (false negatives) or flags different images as duplicates (false positives). At 100k+ images, even 0.1% false positive rate means 100 incorrect matches.



**Why it happens:**

- Single hash algorithm (e.g., only pHash) has blind spots

- Hamming distance threshold is one-size-fits-all

- Anime images with similar composition (same character pose, background) collide more than natural photos

- Edited versions (cropped, watermarked, color-adjusted) fall outside threshold



**How to avoid:**

- Multiple hash algorithms: Combine dHash + pHash + wHash + color hash

- Tiered matching: Exact file hash (SHA-256) for byte-identical, perceptual hash with threshold 5-8 for near-duplicates, AI embedding similarity for same image different edit

- Manual review queue: Do not auto-delete; queue for user confirmation

- Anime-specific: Lower threshold for character portraits (high similarity)



**Warning signs:**

- Users report missing images that were auto-deleted as duplicates

- Different characters with similar poses flagged as same image

- Screenshot collections with minor differences causing explosions



**Phase to address:** Phase 2 (Duplicate Detection)

---



### Pitfall 3: Flutter Image Grid Memory Explosion



**What goes wrong:**

App crashes with OOM (Out of Memory) when scrolling through large image galleries. 1000+ images can consume 500MB+ RAM even with lazy loading.



**Why it happens:**

- GridView.builder without proper image disposal

- Full-resolution images loaded for thumbnails

- No size constraints on Image widget

- CachedNetworkImage default cache too large

- Images not resized before decoding



**How to avoid:**

- Generate thumbnails server-side - never send full-res to grid

- Use ResizeImage widget: ResizeImage(FileImage(file), width: 300)

- Implement proper disposal: cached_network_image with maxAge/maxNrOfCacheObjects

- Memory-aware loading: Limit concurrent image decoding to 3-5

- Recycler pattern: Use ListView.builder with addAutomaticKeepAlives: false

- Hero animations only for detail view, not grid



**Warning signs:**

- App freezes when scrolling fast

- Memory usage climbs and never drops

- Images reload when scrolling back up

- Flutter Image Provider cache is full warnings



**Phase to address:** Phase 3 (Gallery UI), Phase 1 (Thumbnail Generation)

---



### Pitfall 4: AI API Rate Limiting and Cost Explosion



**What goes wrong:**

Scanning a large library (50k+ images) hits API rate limits within hours. Bill surprises from unlimited tiers that have hidden limits. Retry storms make things worse.



**Why it happens:**

- No queuing/throttling for AI requests

- Treating AI API like local function calls

- No exponential backoff on 429 errors

- Processing all images at once on initial import

- Not accounting for API latency (2-5s per image)



**How to avoid:**

- Request queuing: Process images at sustainable rate (e.g., 10/min for free tier)

- Circuit breaker: Pause processing when rate limit hit, resume after window

- Exponential backoff with jitter: retryAfter := baseDelay * (1 << attempt) + rand.Intn(1000)

- Prioritize: Process new images first, backfill old ones during idle

- Local caching: Cache AI results (character tags) to avoid re-processing

- Cost caps: Daily spend limits with alerts



**Warning signs:**

- 429 RESOURCE_EXHAUSTED errors

- API latency increasing (queuing at provider)

- Bill exceeding budget by 10x

- Queue growing faster than processing rate



**Phase to address:** Phase 1 (AI Integration), Phase 4 (Background Processing)

---



### Pitfall 5: Character Recognition False Positives



**What goes wrong:**

AI identifies wrong characters, especially for: similar-looking characters (same hair color, outfit), crossover art (multiple series characters together), low-quality images, partial/cropped faces, fan art with original costumes.



**Why it happens:**

- Anime character faces have high similarity (large eyes, small nose standard style)

- Training data bias toward popular characters

- Same character artist variations not in training set

- Occlusion (hair covering face, accessories) reduces accuracy



**How to avoid:**

- Confidence thresholds: Discard predictions below 70-80%

- Multi-character detection: Do not assume single character per image

- Tag validation: Cross-reference with existing tags (e.g., solo vs multiple_girls)

- User correction: Allow users to flag/correct wrong character tags

- Feedback loop: Use corrections to improve future predictions

- Character aliases: Handle name variations (e.g., Rin vs Tohsaka Rin)



**Warning signs:**

- Character distribution heavily skewed to popular ones

- Users reporting wrong character matches

- Tag confidence scores consistently low

- Same character detected across unrelated images



**Phase to address:** Phase 1 (Character Recognition)

---



### Pitfall 6: Database Schema Too Rigid for Tags



**What goes wrong:**

Tag system cannot evolve: cannot add new tag types (artist, character, general, meta), tag search is slow (LIKE queries scan entire table), no tag hierarchy (cannot search blue_hair and get all hair color tags), tag synonyms not supported (saber vs artoria_pendragon).



**Why it happens:**

- Tags stored as JSON array or comma-separated string

- No dedicated tag table with relationships

- Designed for simple keyword search, not faceted navigation

- Underestimating Danbooru-style tag complexity (5-50 tags per image)



**How to avoid:**

- Separate tables: images, tags, image_tags (junction)

- Tag categories: character, artist, copyright, general, meta

- FTS5 for SQLite: Full-text search with ranking and highlighting

- Tag relationships: parent/child, implication (saber -> fate/stay_night)

- Tag synonyms table: Map user queries to canonical tags

- Indexing: CREATE INDEX idx_image_tags ON image_tags(image_id, tag_id)



**Warning signs:**

- Tag search taking >2 seconds with 10k images

- Cannot implement search within character tags only

- Database migration needed for every new tag type

- Tag autocomplete not working smoothly



**Phase to address:** Phase 1 (Database Schema)

---



### Pitfall 7: EXIF Orientation Ignored



**What goes wrong:**

Images display rotated incorrectly in gallery. Thumbnails show wrong orientation. User confusion about why is my image sideways?



**Why it happens:**

- Reading raw pixel data without checking EXIF Orientation tag

- Thumbnail generator ignores orientation

- Flutter Image widget displays raw bytes without transform



**How to avoid:**

- Read EXIF before processing: Use library that handles orientation (Go: github.com/rwcarlsen/goexif)

- Normalize on import: Rotate image to standard orientation, strip EXIF

- Flutter display: Use RotatedBox or transform based on EXIF

- Consistent handling: Same logic for thumbnails and full images



**Warning signs:**

- Portrait photos showing as landscape

- Thumbnails rotated differently than full image

- iPhone photos consistently wrong orientation



**Phase to address:** Phase 1 (Image Import)

---



### Pitfall 8: Reverse Image Search False Confidence



**What goes wrong:**

Find similar images returns visually unrelated results. Color similarity confused with content similarity. Edited versions not found.



**Why it happens:**

- Using color histograms only (cat vs orange wall both orange)

- Perceptual hash too coarse for fine matching

- Not accounting for aspect ratio changes

- Black bars/borders affect hash significantly



**How to avoid:**

- Multi-signal matching: Color histogram (coarse filter), perceptual hash (medium precision), feature descriptors (fine matching, slower)

- Preprocessing: Strip borders, normalize aspect ratio before hashing

- User feedback: Thumbs up/down to improve results

- Configurable sensitivity: User-adjustable similarity strictness



**Warning signs:**

- Similar image search returns completely different content

- Same image with different aspect ratio not found

- Search results dominated by color matches, not content



**Phase to address:** Phase 3 (Search Implementation)

---



### Pitfall 9: Folder Monitoring Performance



**What goes wrong:**

Folder watcher consumes excessive CPU/I/O. Misses files added during app restart. Duplicate processing of same files.



**Why it happens:**

- Polling instead of OS-level watching (inotify, FSEvents)

- No debouncing (file written in chunks triggers multiple events)

- No persistence of already processed state

- Rescanning entire tree on every startup



**How to avoid:**

- OS-native watching: Go fsnotify library for inotify/FSEvents

- Debounce: Wait 500ms after last write event before processing

- State persistence: Track processed files by hash, not path

- Incremental scanning: Only check mtime > last_scan_time

- Batch processing: Queue files, process in batches every 30s



**Warning signs:**

- High CPU when no user activity

- Files added while app offline never imported

- Same file processed multiple times

- Slow startup with large monitored folders



**Phase to address:** Phase 1 (Folder Monitoring)

---



### Pitfall 10: Go Image Processing Memory Leaks



**What goes wrong:**

Backend memory grows unbounded when processing images. OOM kills during batch thumbnail generation.



**Why it happens:**

- Not closing file handles after opening images

- image.Decode keeping references to large buffers

- Goroutines leaking in worker pools

- No memory limits on concurrent image processing



**How to avoid:**

- Explicit resource cleanup: Always use defer file.Close()

- Worker pool with limits: Max 4-8 concurrent image decodes

- Buffer pooling: Reuse byte slices with sync.Pool

- Memory profiling: Regular runtime.ReadMemStats() monitoring

- Image bounds checking: Reject images >50MB before decoding



**Warning signs:**

- Memory usage grows during batch import

- RSS memory >> actual working set

- GC frequency increasing

- too many open files errors



**Phase to address:** Phase 1 (Image Processing Pipeline)

---



## Technical Debt Patterns



| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |

|----------|-------------------|----------------|-----------------|

| Store full images in DB | Simpler backup (single file) | Database bloat, slow queries, scaling wall | Never for >1000 images |

| Skip thumbnail generation | Faster import | UI lag, memory crashes, poor UX | Only for MVP demo |

| Single hash for duplicates | Simpler code | False positives/negatives at scale | Never for production |

| Process AI tags synchronously | Simpler architecture | Rate limits, timeouts, poor UX | Only with strict queue limits |

| LIKE queries for search | Works immediately | O(n) scan, unusable at 10k+ images | Only until FTS5 implemented |

| No EXIF handling | Skip library dependency | Rotated images forever | Never - fix on import |

| File path as image ID | Simple joins | Breaks when files move | Only with path normalization |

| No image validation | Import anything | Crashes on corrupted files | Never - validate format/magic bytes |

---



## Integration Gotchas



| Integration | Common Mistake | Correct Approach |

|-------------|----------------|------------------|

| AI Character API | Calling API synchronously in request handler | Queue + async processing with retry |

| AI Tagging API | Processing every image on upload | Skip AI for duplicates (use existing tags) |

| AI Rate Limits | Linear retry after 429 | Exponential backoff + jitter + circuit breaker |

| File System Watch | Polling every 5 seconds | OS-native events (fsnotify) with debounce |

| Flutter Image Loading | Loading full-res into grid | Server-side thumbnails + client-side resize |

| SQLite FTS5 | Not using it (LIKE queries) | Create FTS5 virtual table for tag/content search |

| Go Image Decode | Not closing file handles | defer file.Close(), limit concurrent decodes |

| Tag Import | Direct insert on every image | Batch insert with transaction per batch |

---



## Performance Traps



| Trap | Symptoms | Prevention | When It Breaks |

|------|----------|------------|----------------|

| No thumbnail strategy | Grid view lag, OOM | Generate 3 sizes on import | >100 images |

| Full image decode in grid | Memory spikes to 500MB+ | Decode to target size only | >50 images in view |

| Synchronous AI processing | Upload hangs for 5s+ | Queue + async + progress indicator | >10 images/minute |

| Database BLOB storage | Query time >1s | File storage + path in DB | >1000 images |

| LIKE queries for search | Search takes 2s+ | FTS5 index | >5000 tags |

| Single-threaded image scan | Import 1000 images takes hours | Worker pool with 4-8 workers | >100 images |

| No pagination | Loading all image metadata | Cursor-based pagination | >5000 images |

| Unbounded cache | Memory grows indefinitely | LRU cache with size limit | Any cache usage |

| PNG thumbnails | 10x larger than JPEG | Generate JPEG thumbnails | >100 thumbnails |

| No image validation | Crashes on corrupted files | Check magic bytes before decode | First corrupted file |

---



## Security Mistakes



| Mistake | Risk | Prevention |

|---------|------|------------|

| Path traversal in file upload | Attacker reads arbitrary files | Validate path, sanitize filename, chroot jail |

| No image format validation | Server-side RCE via malicious image | Check magic bytes, whitelist formats (jpg, png, gif) |

| EXIF data exposure | GPS location leaks | Strip EXIF on import (except orientation) |

| SQL injection in tag search | Database compromise | Parameterized queries only |

| Unrestricted file upload | Storage DoS | Size limits, extension whitelist |

| SSRF via image URL import | Internal network scanning | Validate URLs, block internal IPs |

| Local file inclusion via path param | Read system files | Validate against whitelist, no ../ |

| No rate limiting on search | Search endpoint DoS | Rate limit per IP/user |

| Thumbnail generation DoS | CPU exhaustion via huge images | Max dimension limits, timeout on processing |

---



## UX Pitfalls



| Pitfall | User Impact | Better Approach |

|---------|-------------|-----------------|

| Blocking UI during import | App frozen, user thinks crashed | Progress bar, cancel button, background processing |

| Auto-delete duplicates silently | User loses images they wanted | Review queue, user confirmation |

| No feedback on AI processing | Tags never appear, user confused | Processing status indicator, pending tags |

| Missing thumbnails show as blank | Grid has empty spaces | Placeholder with filename/loading spinner |

| No undo on bulk operations | Accidental deletion, no recovery | Soft delete, trash folder, restore option |

| Character tags with low confidence | Wrong character labels | Show confidence %, allow user correction |

| No keyboard shortcuts | Power users frustrated | j/k navigation, space for preview, del for delete |

| Infinite scroll without position save | Lose place when returning | Save scroll position, back button works |

| Search with no results is empty page | User confusion | No results with suggestions, recent searches |

| No metadata editing | Wrong tags persist forever | Inline editing, batch tag editing |

---



## Looks Done But Is Not Checklist



- [ ] **Image Import:** Tests with >1000 images, corrupted files, various formats

- [ ] **Thumbnails:** Generated for all images, correct EXIF orientation, reasonable file sizes (<50KB)

- [ ] **Duplicate Detection:** Tested with actual duplicate sets, manual review UI implemented

- [ ] **Character Recognition:** Confidence threshold tuned, multi-character detection, user correction flow

- [ ] **Tag System:** FTS5 working, autocomplete <100ms, tag hierarchy implemented

- [ ] **Folder Monitoring:** Handles app restarts, file moves, debounced events

- [ ] **Search:** Returns relevant results in <500ms, handles typos, no SQL injection

- [ ] **Gallery UI:** Scrolls smoothly with 1000+ images, memory stable, images do not reload

- [ ] **AI Integration:** Queue with retry, rate limit handling, cost tracking

- [ ] **Database:** Migrations tested, backup/restore documented, FTS5 enabled

---



## Recovery Strategies



| Pitfall | Recovery Cost | Recovery Steps |

|---------|---------------|----------------|

| Database BLOB bloat | HIGH | Export metadata, migrate to file storage, regenerate thumbnails |

| Wrong perceptual hash threshold | MEDIUM | Re-scan with new threshold, review false positives/negatives |

| Flutter memory OOM | LOW | Implement thumbnails, add ResizeImage, ship hotfix |

| AI API banned | HIGH | Switch provider, re-tag with new API, handle schema differences |

| Corrupted image DB | HIGH | Restore from backup, re-scan library, re-apply AI tags |

| Character tags wrong | LOW | Bulk delete tags, re-process with higher confidence threshold |

| EXIF orientation ignored | MEDIUM | Regenerate thumbnails with correct rotation, update existing |

| Tag search slow | LOW | Add FTS5 index, update queries to use MATCH |

| Missing folder monitoring events | LOW | Manual rescan, implement state persistence |

| Duplicate false positives | MEDIUM | Review deleted images, restore from trash, tune algorithm |

---



## Pitfall-to-Phase Mapping



| Pitfall | Prevention Phase | Verification |

|---------|------------------|--------------|

| BLOB storage | Phase 1 - Storage | Database size <100MB per 10k images |

| Perceptual hashing | Phase 2 - Deduplication | Test with 1000 actual duplicates, <5% error rate |

| Flutter grid performance | Phase 1 + Phase 3 | Smooth 60fps scroll with 1000 images |

| AI rate limiting | Phase 1 - AI Integration | Process 100 images without 429 errors |

| Character recognition accuracy | Phase 1 - Character ID | User survey: >80% correct identification |

| Tag schema rigidity | Phase 1 - Database | FTS5 search <100ms for any tag query |

| EXIF orientation | Phase 1 - Import | All thumbnails correctly oriented |

| Reverse image search | Phase 3 - Search | Find edited version of 10 test images |

| Folder monitoring | Phase 1 - Monitoring | No missed files during stress test |

| Go memory leaks | Phase 1 - Pipeline | Stable memory during 24h batch import |

---



## Anime/Manga Specific Issues



### Character Recognition Complexity

- **Similar Character Problem:** Anime art style standardization means characters look more alike than real people

- **Crossover Art:** Multiple series characters in one image confuses single-character detection

- **Fan Art Variations:** Artists draw characters in different outfits/poses not in training data

- **Multiple Forms:** Characters with transformations (Magical Girl, Super Saiyan) treated as different



### Tag Complexity (Danbooru-style)

- **Tag Volume:** 5-50 tags per image vs. 3-5 for natural photos

- **Tag Categories:** character, copyright, artist, general, meta - need separate handling

- **Tag Implications:** saber -> fate/stay_night, fate_(series) - need relationship graph

- **Tag Synonyms:** rin vs tohsaka_rin - need disambiguation

- **NSFW Handling:** Rating:safe/questionable/explicit - important for UX



### Image Characteristics

- **Aspect Ratios:** Vertical phone screenshots common (9:16), unlike landscape photos

- **Resolution Variance:** From 400x400 thumbnails to 4000x6000 high-res scans

- **Format Variety:** JPG, PNG, GIF, WebP, sometimes BMP/TIFF from scanners

- **Source Artifacts:** Watermarks, scan borders, compression artifacts from re-saving



### Duplicate Detection Challenges

- **Color Palette Similarity:** Many images share anime color schemes (pastel hair, blue sky)

- **Composition:** Similar poses (character standing, character sitting) cause hash collisions

- **Screenshot Variants:** Same scene with/without subtitles, with different crops

- **Edit Chains:** Original -> resized -> watermarked -> recompressed causes drift from original hash

---



## Sources



1. [The Problem with Perceptual Hashes](https://rentafounder.com/the-problem-with-perceptual-hashes/) - Real-world collision analysis

2. [Detection of (Near) Identical Images Using Image Hash Functions](https://office.qz.com/detection-of-near-identical-images-using-image-hash-functions-c61e133a0958) - Hash comparison techniques

3. [I built a duplicate photo detector that safely cleans 50k+ images](https://www.reddit.com/r/Python/comments/1r73nrb/i_built_a_duplicate_photo_detector_that_safely/) - Practical implementation lessons

4. [The Memory Killer Most Flutter Developers Ignore](https://medium.com/@alaxhenry0121/the-memory-killer-most-flutter-developers-ignore-how-images-are-destroying-your-apps-performance-5c6a50d09e23) - Flutter image performance

5. [Efficient Large List Rendering in Flutter](https://medium.com/@akshatha1729/efficient-large-list-rendering-in-flutter-hash-based-item-identity-done-right-a394c4bbdef5) - List optimization patterns

6. [Fix Gemini Image Rate Limits: 7 Proven Solutions](https://www.aifreeapi.com/en/posts/gemini-image-rate-limit-solution) - AI API rate limiting

7. [SQLite Full-Text Search (FTS5) in Practice](https://thelinuxcode.com/sqlite-full-text-search-fts5-in-practice-fast-search-ranking-and-real-world-patterns/) - Tag search implementation

8. [Go Memory Leaks: Detection, Fixes, and Best Practices](https://medium.com/@mojimich2015/golang-memory-leaks-detection-fixes-and-best-practices-81749e9d698b) - Go memory management

9. [Why Your Database Hates Your Images: A Guide to BLOBs](https://dev.to/hrishikesh_dalal_ced8f95e/system-design-ep-12-why-your-database-hates-your-images-a-guide-to-blobs-2k00) - Database storage patterns

10. [Immich GitHub Issues](https://github.com/immich-app/immich) - Real photo app issues (thumbnail rotation, HEIF support)

11. [Anime Character Identification and Tag Prediction](https://gwern.net/doc/ai/anime/danbooru/2023-yi.pdf) - Academic research on anime tag systems

12. [An Analysis Of Danbooru Tags and Metadata](https://nsk.sh/posts/an-analysis-of-danbooru-tags-and-metadata/) - Tag statistics and patterns



---



*Pitfalls research for: ACGWarehouse (二次元图片库)*

*Researched: 2026-03-14*

---

## v2.0: Dual-Platform UI Pitfalls (Windows Fluent UI + Android Material 3)

**Focus:** Adding Windows desktop and Android mobile UI to existing Flutter Web app
**Researched:** 2026-03-20
**Confidence:** HIGH (Context7 + Flutter official docs + GitHub code patterns)

---

### Critical UI Pitfalls

### Pitfall 11: Dual MaterialApp/FluentApp Root Widget

**What goes wrong:**
Developers try to nest `MaterialApp` inside `FluentApp` (or vice versa), causing theme context issues, navigation problems, and duplicate app wrappers. Widgets lose access to their respective theme data, resulting in "No MaterialLocalizations found" or "No FluentLocalizations found" errors.

**Why it happens:**
Both `MaterialApp` and `FluentApp` are wrapper widgets that provide navigation, theming, and localization. Nesting them creates conflicting contexts and breaks the theme inheritance chain.

**How to avoid:**
- Use a single platform-aware root widget that conditionally returns `MaterialApp` or `FluentApp`
- Extract shared navigation logic into a separate layer (Provider state + route definitions)
- Never nest `MaterialApp` inside `FluentApp` or vice versa

**Warning signs:**
- "No MaterialLocalizations found" errors
- Theme.of(context) returns null or wrong theme
- Navigation routes work on one platform but not another

**Phase to address:** Phase 1 (Architecture Foundation)

---

### Pitfall 12: Hardcoded Responsive Breakpoints

**What goes wrong:**
Developers use fixed pixel values (e.g., `width > 600`) without considering device pixel ratio, safe areas, or orientation changes. Layouts break on tablets, foldables, or when the window resizes.

**Why it happens:**
Direct `MediaQuery.of(context).size.width` comparisons seem simple but don't account for:
- Device pixel ratio differences
- System UI overlays (status bar, navigation bar)
- Foldable device hinge areas
- Window resizing on desktop

**How to avoid:**
- Use Flutter's Material breakpoints: 600 (phone/tablet), 840 (tablet/desktop)
- Wrap responsive logic in `LayoutBuilder` for constraint-based decisions
- Consider `MediaQuery.of(context).padding` for safe areas
- Test on multiple form factors during development

**Warning signs:**
- UI breaks when rotating device
- Content cut off on certain devices
- Desktop window resize causes layout overflow

**Phase to address:** Phase 2 (Responsive Layout System)

---

### Pitfall 13: Mismatched Navigation Patterns

**What goes wrong:**
Using `NavigationBar` (bottom) on desktop or `NavigationRail` (side) on small phones creates poor UX. Navigation state management becomes complex when switching between navigation types.

**Why it happens:**
- `NavigationRail` takes vertical space, poor fit for narrow screens
- `NavigationBar` at bottom wastes desktop vertical space
- State index synchronization between different navigation widgets is error-prone

**How to avoid:**
- Use platform conventions: `NavigationRail` for width >= 600, `NavigationBar` for width < 600
- For Windows, use `NavigationView` with `PaneDisplayMode.auto` (auto-adapts)
- Centralize navigation state in Provider, let UI components observe it
- Test navigation state preservation when resizing window

**Warning signs:**
- Navigation state resets when rotating device
- Selected tab doesn't highlight correctly on platform switch
- Deep links don't work consistently across platforms

**Phase to address:** Phase 3 (Adaptive Navigation)

---

### Pitfall 14: Theme Data Type Confusion

**What goes wrong:**
Calling `Theme.of(context)` inside Fluent UI widgets returns Material theme data. Calling `FluentTheme.of(context)` inside Material widgets throws errors or returns null.

**Why it happens:**
`MaterialApp` provides `ThemeData` through `InheritedWidget`. `FluentApp` provides `FluentThemeData`. These are separate type hierarchies and cannot be interchanged.

**How to avoid:**
- Create platform-specific theme wrappers
- Use a unified "app theme" Provider that exposes semantic properties (colors, text styles) as abstract getters
- Platform UI widgets consume from the unified theme Provider

**Warning signs:**
- Colors don't match between platforms
- Typography looks different on Windows vs Android
- Theme changes don't propagate to all widgets

**Phase to address:** Phase 4 (Custom Design System)

---

### Pitfall 15: Platform.isX Checks in Widget Build

**What goes wrong:**
Using `Platform.isWindows` directly in `build()` causes crashes on web (Platform not available) and makes testing difficult. The widget tree rebuilds incorrectly when platform detection changes.

**Why it happens:**
- `dart:io` Platform class is not available on web
- Widget build methods should be pure and deterministic
- Platform detection should happen once, not every build

**How to avoid:**
- Use `defaultTargetPlatform` instead of `Platform.isX` for UI decisions
- Create platform detection at app initialization
- For web compatibility, wrap Platform checks in `kIsWeb` guard

**Warning signs:**
- App crashes on web with "Unsupported operation: Platform._operatingSystem"
- Hot reload causes platform detection to fail
- Tests can't mock platform

**Phase to address:** Phase 1 (Architecture Foundation)

---

### Pitfall 16: State Loss During Navigation Switch

**What goes wrong:**
When switching from `NavigationRail` to `NavigationBar` (or vice versa during resize), the current screen's state (scroll position, form data, selected items) is lost because the widget tree is completely rebuilt.

**Why it happens:**
Different navigation patterns create different widget hierarchies. Without proper state preservation, switching navigation types triggers `dispose()` on child widgets.

**How to avoid:**
- Keep navigation state in Provider/Bloc (above the navigation widget)
- Use `PageStorage` for scroll position preservation
- Consider `IndexedStack` for preserving all screens in memory
- Extract screen content into stateless widgets that receive state from Provider

**Warning signs:**
- Scroll position resets when resizing window
- Form data lost when switching to landscape
- Selected grid items deselect on orientation change

**Phase to address:** Phase 3 (Adaptive Navigation)

---

## v2.0 Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Copy-paste widgets for each platform | Faster initial dev | Duplicate code, double maintenance | Never |
| Skip responsive breakpoints | Ship faster | Breaks on tablets, foldables | Prototype only |
| Hardcode colors in widgets | Quick styling | Theme changes require widget edits | Never |
| Use Platform.isWindows everywhere | Platform checks work | Scattered platform logic, hard to test | Never |
| Ignore accessibility (screen readers) | Save time | Legal/compliance issues, excludes users | Never |
| Single column layout for all sizes | Simple implementation | Wasted space on desktop | MVP only |

---

## v2.0 Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| fluent_ui + Provider | Accessing Provider in NavigationView.body without wrapper | Provider must wrap FluentApp or be at higher level |
| NavigationPane -> Navigator | Using Navigator.push() inside NavigationPane body | NavigationPane already manages body, use indexed navigation |
| Material widgets in FluentApp | Using Scaffold inside NavigationView | Use ScaffoldPage or Page from fluent_ui |
| FluentThemeData <-> ThemeData | Trying to convert between theme types | Create unified theme interface, implement per-platform |
| showDialog in Fluent context | Using Material showDialog | Use showDialog from fluent_ui which returns ContentDialog |
| Flutter window manager | Not handling Windows minimize/maximize | Add window_manager package for native window controls |

---

## v2.0 Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| LayoutBuilder rebuild storm | Janky scrolling, high CPU | Memoize compu

---

## v2.0: Dual-Platform UI Pitfalls (Windows Fluent UI + Android Material 3)

**Focus:** Adding Windows desktop and Android mobile UI to existing Flutter Web app
**Researched:** 2026-03-20
**Confidence:** HIGH (Context7 + Flutter official docs + GitHub code patterns)

---

### Critical UI Pitfalls

### Pitfall 11: Dual MaterialApp/FluentApp Root Widget

**What goes wrong:**
Developers try to nest `MaterialApp` inside `FluentApp` (or vice versa), causing theme context issues, navigation problems, and duplicate app wrappers. Widgets lose access to their respective theme data, resulting in "No MaterialLocalizations found" or "No FluentLocalizations found" errors.

**Why it happens:**
Both `MaterialApp` and `FluentApp` are wrapper widgets that provide navigation, theming, and localization. Nesting them creates conflicting contexts and breaks the theme inheritance chain.

**How to avoid:**
- Use a single platform-aware root widget that conditionally returns `MaterialApp` or `FluentApp`
- Extract shared navigation logic into a separate layer (Provider state + route definitions)
- Never nest `MaterialApp` inside `FluentApp` or vice versa

**Warning signs:**
- "No MaterialLocalizations found" errors
- Theme.of(context) returns null or wrong theme
- Navigation routes work on one platform but not another

**Phase to address:** Phase 1 (Architecture Foundation)

---

### Pitfall 12: Hardcoded Responsive Breakpoints

**What goes wrong:**
Developers use fixed pixel values (e.g., `width > 600`) without considering device pixel ratio, safe areas, or orientation changes. Layouts break on tablets, foldables, or when the window resizes.

**Why it happens:**
Direct `MediaQuery.of(context).size.width` comparisons seem simple but don't account for:
- Device pixel ratio differences
- System UI overlays (status bar, navigation bar)
- Foldable device hinge areas
- Window resizing on desktop

**How to avoid:**
- Use Flutter's Material breakpoints: 600 (phone/tablet), 840 (tablet/desktop)
- Wrap responsive logic in `LayoutBuilder` for constraint-based decisions
- Consider `MediaQuery.of(context).padding` for safe areas
- Test on multiple form factors during development

**Warning signs:**
- UI breaks when rotating device
- Content cut off on certain devices
- Desktop window resize causes layout overflow

**Phase to address:** Phase 2 (Responsive Layout System)

---

### Pitfall 13: Mismatched Navigation Patterns

**What goes wrong:**
Using `NavigationBar` (bottom) on desktop or `NavigationRail` (side) on small phones creates poor UX. Navigation state management becomes complex when switching between navigation types.

**Why it happens:**
- `NavigationRail` takes vertical space, poor fit for narrow screens
- `NavigationBar` at bottom wastes desktop vertical space
- State index synchronization between different navigation widgets is error-prone

**How to avoid:**
- Use platform conventions: `NavigationRail` for width >= 600, `NavigationBar` for width < 600
- For Windows, use `NavigationView` with `PaneDisplayMode.auto` (auto-adapts)
- Centralize navigation state in Provider, let UI components observe it
- Test navigation state preservation when resizing window

**Warning signs:**
- Navigation state resets when rotating device
- Selected tab doesn't highlight correctly on platform switch
- Deep links don't work consistently across platforms

**Phase to address:** Phase 3 (Adaptive Navigation)

---

### Pitfall 14: Theme Data Type Confusion

**What goes wrong:**
Calling `Theme.of(context)` inside Fluent UI widgets returns Material theme data. Calling `FluentTheme.of(context)` inside Material widgets throws errors or returns null.

**Why it happens:**
`MaterialApp` provides `ThemeData` through `InheritedWidget`. `FluentApp` provides `FluentThemeData`. These are separate type hierarchies and cannot be interchanged.

**How to avoid:**
- Create platform-specific theme wrappers
- Use a unified "app theme" Provider that exposes semantic properties (colors, text styles) as abstract getters
- Platform UI widgets consume from the unified theme Provider

**Warning signs:**
- Colors don't match between platforms
- Typography looks different on Windows vs Android
- Theme changes don't propagate to all widgets

**Phase to address:** Phase 4 (Custom Design System)

---

### Pitfall 15: Platform.isX Checks in Widget Build

**What goes wrong:**
Using `Platform.isWindows` directly in `build()` causes crashes on web (Platform not available) and makes testing difficult. The widget tree rebuilds incorrectly when platform detection changes.

**Why it happens:**
- `dart:io` Platform class is not available on web
- Widget build methods should be pure and deterministic
- Platform detection should happen once, not every build

**How to avoid:**
- Use `defaultTargetPlatform` instead of `Platform.isX` for UI decisions
- Create platform detection at app initialization
- For web compatibility, wrap Platform checks in `kIsWeb` guard

**Warning signs:**
- App crashes on web with "Unsupported operation: Platform._operatingSystem"
- Hot reload causes platform detection to fail
- Tests can't mock platform

**Phase to address:** Phase 1 (Architecture Foundation)

---

### Pitfall 16: State Loss During Navigation Switch

**What goes wrong:**
When switching from `NavigationRail` to `NavigationBar` (or vice versa during resize), the current screen's state (scroll position, form data, selected items) is lost because the widget tree is completely rebuilt.

**Why it happens:**
Different navigation patterns create different widget hierarchies. Without proper state preservation, switching navigation types triggers `dispose()` on child widgets.

**How to avoid:**
- Keep navigation state in Provider/Bloc (above the navigation widget)
- Use `PageStorage` for scroll position preservation
- Consider `IndexedStack` for preserving all screens in memory
- Extract screen content into stateless widgets that receive state from Provider

**Warning signs:**
- Scroll position resets when resizing window
- Form data lost when switching to landscape
- Selected grid items deselect on orientation change

**Phase to address:** Phase 3 (Adaptive Navigation)

---

## v2.0 Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Copy-paste widgets for each platform | Faster initial dev | Duplicate code, double maintenance | Never |
| Skip responsive breakpoints | Ship faster | Breaks on tablets, foldables | Prototype only |
| Hardcode colors in widgets | Quick styling | Theme changes require widget edits | Never |
| Use `!kIsWeb && Platform.isWindows` everywhere | Platform checks work | Scattered platform logic, hard to test | Never |
| Ignore accessibility (screen readers) | Save time | Legal/compliance issues, excludes users | Never |
| Single column layout for all sizes | Simple implementation | Wasted space on desktop | MVP only |

---

## v2.0 Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| fluent_ui + Provider | Accessing Provider in `NavigationView.body` without wrapper | Provider must wrap `FluentApp` or be at higher level |
| NavigationPane -> Navigator | Using `Navigator.push()` inside NavigationPane body | NavigationPane already manages body, use indexed navigation |
| Material widgets in FluentApp | Using `Scaffold` inside `NavigationView` | Use `ScaffoldPage` or `Page` from fluent_ui |
| FluentThemeData <-> ThemeData | Trying to convert between theme types | Create unified theme interface, implement per-platform |
| showDialog in Fluent context | Using Material `showDialog` | Use `showDialog` from fluent_ui which returns `ContentDialog` |
| Flutter window manager | Not handling Windows minimize/maximize | Add `window_manager` package for native window controls |

---

## v2.0 Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| LayoutBuilder rebuild storm | Janky scrolling, high CPU | Memoize computed layouts, use `const` widgets | Large grid views |
| Navigation index rebuilds all | Switching tabs is slow | Use `IndexedStack` or `PageView` with `keepPage: true` | 5+ navigation items |
| Unoptimized image loading | Memory spikes, lag | Use `cached_network_image`, implement pagination | 100+ images displayed |
| Provider notifyListeners spam | UI stutters on updates | Batch updates, use `select` for granular rebuilds | Real-time updates |
| MediaQuery rebuild cascade | Every pixel resize rebuilds tree | Cache MediaQuery values, use `LayoutBuilder` | Desktop window resize |

---

## v2.0 UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Bottom nav on desktop | Wastes vertical space, feels "mobile" | Use side navigation (NavigationRail/NavigationView) |
| No loading states during resize | Blank screen, confusing | Show skeleton or shimmer during layout transition |
| Different navigation order per platform | Users lose mental model | Keep navigation items in same order, just different layout |
| Touch targets too small on desktop | Hard to click with mouse | Minimum 44x44 on mobile, consider larger on desktop |
| No keyboard shortcuts on Windows | Power users frustrated | Add keyboard shortcuts, accelerator keys in menus |
| Ignoring system theme (dark mode) | Jarring bright/dark mismatch | Use `ThemeMode.system` or `FluentThemeMode.system` |

---

## v2.0 "Looks Done But Isn't" Checklist

- [ ] **Responsive Layout:** Often missing tablet/desktop specific layouts - verify on 7", 10", 13" screens
- [ ] **Navigation State:** Often missing state preservation on resize - verify scroll position retained
- [ ] **Theme Sync:** Often missing dark mode support on both platforms - verify theme toggle works everywhere
- [ ] **Platform Icons:** Often missing platform-appropriate icons - verify Windows uses FluentIcons, Android uses Material icons
- [ ] **Dialogs/Modals:** Often missing platform-specific dialogs - verify `showDialog` uses correct style per platform
- [ ] **Keyboard Navigation:** Often missing Tab/Enter navigation on Windows - verify keyboard accessibility
- [ ] **Window Controls:** Often missing native minimize/maximize/close - verify `window_manager` integration
- [ ] **Safe Areas:** Often missing notch/status bar handling - verify content not obscured by system UI

---

## v2.0 Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Dual MaterialApp/FluentApp | HIGH | Refactor to single conditional root, rebuild navigation architecture |
| Hardcoded breakpoints | MEDIUM | Extract to responsive utilities, add LayoutBuilder wrappers |
| Mismatched navigation | MEDIUM | Create unified navigation state, refactor navigation widgets |
| Theme type confusion | HIGH | Create unified theme interface, refactor all color references |
| Platform.isX in build | LOW | Replace with `defaultTargetPlatform`, add web guards |
| State loss on resize | MEDIUM | Add Provider state layer, implement PageStorage |

---

## v2.0 Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Dual MaterialApp/FluentApp | Phase 1 (Architecture) | Run app on Windows and Android, verify no context errors |
| Hardcoded breakpoints | Phase 2 (Responsive) | Test on phone, tablet, desktop window resize |
| Mismatched navigation | Phase 3 (Navigation) | Verify navigation state preserved on orientation change |
| Theme type confusion | Phase 4 (Design System) | Toggle theme, verify both platforms update consistently |
| Platform.isX in build | Phase 1 (Architecture) | Run flutter analyze, test on web platform |
| State loss on resize | Phase 3 (Navigation) | Scroll, resize window, verify scroll position retained |

---

## v2.0 Sources

- Context7: fluent_ui documentation (/bdlukaa/fluent_ui) - HIGH confidence
- Context7: Flutter official documentation (/websites/flutter_dev) - HIGH confidence
- GitHub code patterns: NavigationRail, NavigationBar, PaneDisplayMode usage - MEDIUM confidence
- Flutter platform adaptation guide - HIGH confidence
- Material Design responsive guidelines - HIGH confidence

---

*Pitfalls research for: ACGWarehouse (Anime Image Gallery)*
*Researched: 2026-03-14 (v1.0), 2026-03-20 (v2.0 UI)*