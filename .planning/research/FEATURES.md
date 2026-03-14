# Feature Research: ACGWarehouse (二次元图片库)

**Domain:** Anime/Manga Image Library Management System
**Researched:** 2026-03-14
**Confidence:** HIGH

## Executive Summary

ACGWarehouse targets anime/manga image collectors who need to manage large local collections. The market has established players (Hydrus, ImoutoRebirth, FEMBOY) but none dominate. Key differentiation opportunities exist in: modern UI/UX, AI-powered automation, and "存入即整理" (store-and-organize) experience.

---

## Feature Landscape

### Table Stakes (Must-Have)

Features users assume exist. Missing these = product feels broken.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Image Import/Scanning | Users have existing folders to ingest | MEDIUM | Folder monitoring, batch import, format support (jpg/png/webp/gif) |
| Basic File Browsing | Need to view what is in the library | LOW | Grid view, thumbnail generation, basic navigation |
| Search (Text) | Find images by filename/basic metadata | MEDIUM | Filename search, simple filters |
| Folder/Album Organization | Users expect to group images | LOW | Collections, albums, virtual folders |
| Duplicate Detection | Large collections have duplicates | MEDIUM | Perceptual hashing, similarity threshold |
| Basic Metadata Storage | Store filename, path, import date | LOW | EXIF, file stats, import timestamp |

### Differentiators (Competitive Advantage)

Features that set ACGWarehouse apart. Not required, but valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| AI Character Recognition | Auto-identify anime characters without manual tagging | HIGH | Requires AI service integration; key value prop |
| Auto-Tagging (Artist/Original/Source) | Extract creator, series, style automatically | HIGH | Booru API integration or AI models; reduces manual work significantly |
| Similar Image Detection | Find near-duplicates and variations | HIGH | Feature vector comparison; memory intensive |
| Reverse Image Search (Internal) | Search own library by uploading an image | HIGH | Requires image embedding/indexing; very useful for large collections |
| Booru Tag Sync | Fetch/update tags from Danbooru/Gelbooru/etc | MEDIUM | API integration, tag conflict resolution |
| Tag-Based Browsing | Browse by character, artist, series, etc | MEDIUM | Faceted search, tag clouds, related tags |
| Smart Collections | Auto-populated albums based on rules | MEDIUM | Query-based collections that update automatically |
| Cross-Platform Client | Access from desktop + mobile | HIGH | Flutter enables this; competitors often desktop-only |
| Folder Monitoring (Auto-Import) | Watch folders, auto-process new images | MEDIUM | File system watchers, background processing |


### Anti-Features (Deliberately NOT Building)

Features that seem good but create problems or do not fit the product vision.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Social Features (Sharing/Following) | Users want to share collections | Violates privacy-first personal library positioning; complexity explosion | Export/share individual images only |
| Cloud Sync/Storage | Access from anywhere | Cost, complexity, privacy concerns | Local network access via Flutter app |
| Video Management | Some users have anime clips | Scope creep; completely different metadata needs | Out of scope per PROJECT.md |
| Image Editing | Users want to crop/resize | Not core value prop; OS tools exist better | Integration with external editors only |
| Built-in Web Browser/Downloader | Hydrus has this for booru scraping | Legal gray area, maintenance burden, focus drift | Manual import workflow |
| Plugin System (Early) | Power users want extensibility | Premature abstraction, API instability | Build core features first, consider v2+ |
| Multiple User Accounts | Multi-user household scenarios | Adds auth complexity, not primary use case | Single-user with optional basic auth |

### Anime/Manga Specific Features

Specialized features for the ACG domain that general image libraries do not have.

| Feature | Why It Matters | Complexity | Notes |
|---------|----------------|------------|-------|
| Booru-Style Tag System | Anime community uses specific tag taxonomies (character, artist, copyright, general) | MEDIUM | Need tag types, hierarchies, aliases |
| Pixiv Integration | Primary source for anime artwork | MEDIUM | API rate limits, requires auth |
| Character Database | Link to character info (AniList/MyAnimeList) | MEDIUM | Character name normalization, aliases |
| Source Attribution | Track where image came from (Pixiv ID, Twitter handle, etc) | LOW | URL metadata, source detection |
| Rating/Content Filter | NSFW/SFW content separation | LOW | Tag-based filtering, user preference |
| Pixiv Ugoira Support | Animated illustrations popular in anime community | MEDIUM | ZIP-based animation format |
| Notes/Translation Overlay | Boorus have translation notes on images | HIGH | Coordinate-based overlay system |


---

## Feature Dependencies

```
[Image Import]
    |--requires--> [File Storage]
    |                 |--requires--> [Thumbnail Generation]
    |--requires--> [Duplicate Detection]

[AI Character Recognition]
    |--requires--> [Image Import]
    |--requires--> [Character Database]

[Auto-Tagging]
    |--requires--> [Image Import]
    |--requires--> [Booru API Integration]
                        |--requires--> [Tag System]

[Reverse Image Search]
    |--requires--> [Image Embedding/Indexing]
                        |--requires--> [Image Import]

[Smart Collections]
    |--requires--> [Search System]
                        |--requires--> [Tag System]

[Tag-Based Browsing]
    |--requires--> [Tag System]
                        |--requires--> [Image Import]

[Booru Tag Sync]
    |--requires--> [Tag System]
    |--conflicts--> [AI Auto-Tagging] (need tag priority rules)
```

### Dependency Notes

- **AI Character Recognition requires Character Database**: Need canonical character names to store recognition results
- **Auto-Tagging requires Booru API or AI Models**: External dependency for tag generation
- **Reverse Image Search requires Image Indexing**: Need vector embeddings for similarity search
- **Booru Tag Sync conflicts with AI Auto-Tagging**: When both provide tags for same image, need merge/priority strategy
- **Smart Collections requires Search System**: Collections are saved search queries that auto-update

---

## MVP Definition

### Launch With (v1)

Minimum viable product - what is needed to validate the concept.

- [x] **Image Scanning & Import** - Core entry point; users need to get images in
- [x] **Basic Gallery View** - Grid/waterfall layout, thumbnails, basic navigation
- [x] **Folder Monitoring** - Auto-import from watched folders; critical for workflow
- [x] **Duplicate Detection** - Table stakes for any serious image library
- [x] **Basic Search** - Filename and simple metadata search
- [x] **Album/Collection Management** - Manual organization users expect

### Add After Validation (v1.x)

Features to add once core is working and validated.

- [ ] **AI Character Recognition** - Key differentiator; requires AI service
- [ ] **Auto-Tagging (Artist/Source)** - Reduces manual work significantly
- [ ] **Tag-Based Browsing** - Requires tag system + populated tags first
- [ ] **Booru Tag Sync** - Fetch tags from Danbooru/Gelbooru
- [ ] **Reverse Image Search** - Search own library by image

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] **Similar Image Detection** - Beyond exact duplicates, find variations
- [ ] **Smart Collections** - Auto-updating rule-based albums
- [ ] **Mobile Sync** - Selective sync to mobile devices
- [ ] **Notes/Translation Overlay** - Display booru notes on images
- [ ] **Plugin System** - Extensibility for power users
- [ ] **WebDAV/FUSE Integration** - Mount library as filesystem


---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Image Scanning & Import | HIGH | MEDIUM | P1 |
| Basic Gallery View | HIGH | LOW | P1 |
| Folder Monitoring | HIGH | MEDIUM | P1 |
| Duplicate Detection | HIGH | MEDIUM | P1 |
| Basic Search | HIGH | LOW | P1 |
| Album Management | MEDIUM | LOW | P1 |
| AI Character Recognition | HIGH | HIGH | P2 |
| Auto-Tagging | HIGH | HIGH | P2 |
| Tag-Based Browsing | MEDIUM | MEDIUM | P2 |
| Booru Tag Sync | MEDIUM | MEDIUM | P2 |
| Reverse Image Search | HIGH | HIGH | P2 |
| Similar Image Detection | MEDIUM | HIGH | P3 |
| Smart Collections | MEDIUM | MEDIUM | P3 |
| Mobile Sync | MEDIUM | HIGH | P3 |
| Notes Overlay | LOW | HIGH | P3 |

**Priority Key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

---

## Competitor Feature Analysis

| Feature | Hydrus | ImoutoRebirth | FEMBOY | Mangatsu | Our Approach |
|---------|--------|---------------|--------|----------|--------------|
| Auto-Tagging | Parser-based download | Booru API sync | DeepDanbooru AI | Manual only | AI + Booru hybrid |
| UI/UX | Desktop Qt (dated) | Desktop WPF (Windows) | Desktop + Mobile | Web-based | Flutter (modern, cross-platform) |
| Character Recognition | No | No | No | No | Yes AI-powered (differentiator) |
| Reverse Search | No | No | No | No | Yes Internal library search |
| Tag Sync | Parser-based | Real-time Booru sync | No | No | On-demand Booru sync |
| Cross-Platform | Win/Linux/Mac | Windows only | All platforms | Web (all) | All platforms (Flutter) |
| Folder Monitoring | Yes | Yes | No | No | Yes Real-time monitoring |
| Collection Focus | General media | Anime images | Anime images | Manga/Doujin | Anime images (optimized) |

### Competitive Positioning

**Hydrus**: Powerful but dated UI, steep learning curve. Targets power users.

**ImoutoRebirth**: Windows-only, Booru-centric. Good for existing booru users.

**FEMBOY**: Mobile+Desktop, AI tagging. Limited active development.

**Mangatsu**: Manga-focused, web-based. Different media type.

**ACGWarehouse Opportunity**: Modern Flutter UI + AI automation + cross-platform + anime-optimized workflow.


---

## Technical Complexity Notes

### High Complexity Features

1. **AI Character Recognition**: Requires training data, model inference infrastructure, character database maintenance
2. **Reverse Image Search**: Needs vector database (Milvus/Pinecone), embedding generation, similarity indexing
3. **Similar Image Detection**: Perceptual hashing at scale, threshold tuning, false positive management
4. **Cross-Platform Sync**: Conflict resolution, bandwidth optimization, selective sync logic

### Medium Complexity Features

1. **Auto-Tagging**: External API integration, rate limiting, tag normalization
2. **Booru Tag Sync**: API auth, tag type mapping, update conflict handling
3. **Folder Monitoring**: File system watchers, event debouncing, move detection
4. **Smart Collections**: Query parser, cache invalidation, background updates

### Low Complexity Features

1. **Basic Gallery**: Thumbnail generation, pagination, view modes
2. **Album Management**: CRUD operations, many-to-many relationships
3. **Basic Search**: Full-text index, simple filters
4. **Duplicate Detection**: Perceptual hash library integration

---

## Sources

### Competitor Products Analyzed

- [Hydrus Network](https://github.com/hydrusnetwork/hydrus) - Personal booru-style media tagger
- [ImoutoRebirth](https://github.com/ImoutoChan/ImoutoRebirth) - Anime image storage with Booru sync
- [FEMBOY](https://github.com/k1rak1ra/FEMBOY) - DeepDanbooru-based auto-tagging system
- [Mangatsu](https://github.com/Mangatsu/server) - Manga/doujinshi media server (Go)
- [Shoko](https://shokoanime.com/) - Anime video management system
- [Seanime](https://seanime.app/) - Anime/manga streaming media server

### Reverse Image Search Services

- [SauceNAO](https://saucenao.com/) - Anime/manga artwork source finder
- [trace.moe](https://trace.moe/) - Anime scene search from screenshot
- [IQDB](https://iqdb.org/) - Multi-service image search aggregator
- [SmartImage](https://github.com/Decimation/SmartImage) - Multi-engine reverse search tool

### Booru Systems Referenced

- [Danbooru](https://danbooru.donmai.us/) - Original booru, tag taxonomy reference
- [Gelbooru](https://gelbooru.com/) - Popular anime imageboard
- [DeepDanbooru](https://github.com/KichangKim/DeepDanbooru) - AI-based anime image tagger

### General Image Management

- [PhotoPrism](https://www.photoprism.app/) - Self-hosted photo management
- [Immich](https://immich.app/) - Self-hosted photo backup solution
- [DigiKam](https://www.digikam.org/) - Professional photo management

---

## Appendix: Booru Tag Taxonomy

Standard tag types used by Danbooru and derived systems:

| Tag Type | Description | Examples |
|----------|-------------|----------|
| Character | Fictional characters depicted | hatsune_miku, naruto |
| Artist | Creator of the artwork | kyo_(kuroichigo) |
| Copyright | Source material (anime/game) | touhou, pokemon |
| General | Visual descriptors | blonde_hair, sunset, smile |
| Meta | Information about the post | highres, commentary, translated |

This taxonomy is essential for organizing anime images effectively and is expected by users familiar with booru culture.

---

*Feature research for: ACGWarehouse (二次元图片库)*
*Researched: 2026-03-14*
*Based on: PROJECT.md requirements + competitor analysis*
