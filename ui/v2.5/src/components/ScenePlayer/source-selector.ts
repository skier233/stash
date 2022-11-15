import videojs, { VideoJsPlayer } from "video.js";

export interface ISource extends videojs.Tech.SourceObject {
  label?: string;
}

class SourceMenuItem extends videojs.getComponent("MenuItem") {
  public source: ISource;
  public isSelected = false;

  constructor(parent: SourceMenuButton, source: ISource) {
    const options = {} as videojs.MenuItemOptions;
    options.selectable = true;
    options.multiSelectable = false;
    options.label = source.label || source.type;

    super(parent.player(), options);

    this.source = source;

    this.addClass("vjs-source-menu-item");
  }

  selected(selected: boolean): void {
    super.selected(selected);
    this.isSelected = selected;
  }

  handleClick() {
    if (this.isSelected) return;

    this.trigger("selected");
  }
}

class SourceMenuButton extends videojs.getComponent("MenuButton") {
  private items: SourceMenuItem[] = [];
  private selectedSource: ISource | null = null;

  constructor(player: VideoJsPlayer) {
    super(player);

    player.on("loadstart", () => {
      this.update();
    });
  }

  public setSources(sources: ISource[]) {
    this.selectedSource = null;

    this.items = sources.map((source, i) => {
      if (i === 0) {
        this.selectedSource = source;
      }

      const item = new SourceMenuItem(this, source);

      item.on("selected", () => {
        this.selectedSource = source;

        this.trigger("sourceselected", source);
      });

      return item;
    });
  }

  createEl() {
    return videojs.dom.createEl("div", {
      className:
        "vjs-source-selector vjs-menu-button vjs-menu-button-popup vjs-control vjs-button",
    });
  }

  createItems() {
    if (this.items === undefined) return [];

    for (const item of this.items) {
      item.selected(item.source === this.selectedSource);
    }

    return this.items;
  }
}

class SourceSelectorPlugin extends videojs.getPlugin("plugin") {
  private menu: SourceMenuButton;
  private sources: ISource[] = [];
  private selectedIndex = -1;
  private cleanupTextTracks: HTMLTrackElement[] = [];
  private manualTextTracks: HTMLTrackElement[] = [];

  constructor(player: VideoJsPlayer) {
    super(player);

    this.menu = new SourceMenuButton(player);

    this.menu.on("sourceselected", (_, source: ISource) => {
      this.selectedIndex = this.sources.findIndex((src) => src === source);
      if (this.selectedIndex === -1) return;

      const currentTime = player.currentTime();

      // put the selected source at the top of the list
      const loadSources = [...this.sources];
      const selectedSrc = loadSources.splice(this.selectedIndex, 1)[0];
      loadSources.unshift(selectedSrc);

      const paused = player.paused();
      player.src(loadSources);
      player.one("canplay", () => {
        if (paused) {
          player.pause();
        }
        player.currentTime(currentTime);
      });
      player.play();
    });

    player.on("ready", () => {
      const { controlBar } = player;
      const fullscreenToggle = controlBar.getChild("fullscreenToggle")!.el();
      controlBar.addChild(this.menu);
      controlBar.el().insertBefore(this.menu.el(), fullscreenToggle);
    });

    player.on("loadedmetadata", () => {
      if (!player.videoWidth() && !player.videoHeight()) {
        // Occurs during preload when videos with supported audio/unsupported video are preloaded.
        // Treat this as a decoding error and try the next source without playing.
        // However on Safari we get an media event when m3u8 is loaded which needs to be ignored.
        if (player.error() !== null) return;
        const currentSrc = player.currentSrc();
        if (currentSrc !== null && !currentSrc.includes(".m3u8")) {
          player.error(MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED);
          return;
        }
      }
    });

    player.on("error", () => {
      const error = player.error();
      if (!error) return;

      // Only try next source if media was unsupported
      if (error.code !== MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED) return;

      const currentSource = player.currentSource() as ISource;
      console.log(`Source '${currentSource.label}' is unsupported`);

      if (this.sources.length > 1) {
        if (this.selectedIndex === -1) return;

        this.sources.splice(this.selectedIndex, 1);
        const newSource = this.sources[0];
        console.log(`Trying next source in playlist: '${newSource.label}'`);
        this.menu.setSources(this.sources);
        this.selectedIndex = 0;
        player.src(this.sources);
        player.load();
        player.play();
      } else {
        console.log("No more sources in playlist");
      }
    });
  }

  setSources(sources: ISource[]) {
    const cleanupTracks = this.cleanupTextTracks.splice(0);
    for (const track of cleanupTracks) {
      this.player.removeRemoteTextTrack(track);
    }

    this.menu.setSources(sources);
    if (sources.length !== 0) {
      this.selectedIndex = 0;
    } else {
      this.selectedIndex = -1;
    }

    this.sources = sources;
    this.player.src(this.sources);
  }

  get textTracks(): HTMLTrackElement[] {
    return [...this.cleanupTextTracks, ...this.manualTextTracks];
  }

  addTextTrack(options: videojs.TextTrackOptions, manualCleanup: boolean) {
    const track = this.player.addRemoteTextTrack(options, true);
    if (manualCleanup) {
      this.manualTextTracks.push(track);
    } else {
      this.cleanupTextTracks.push(track);
    }
    return track;
  }

  removeTextTrack(track: HTMLTrackElement) {
    this.player.removeRemoteTextTrack(track);
    let index = this.manualTextTracks.indexOf(track);
    if (index != -1) {
      this.manualTextTracks.splice(index, 1);
    }
    index = this.cleanupTextTracks.indexOf(track);
    if (index != -1) {
      this.cleanupTextTracks.splice(index, 1);
    }
  }
}

// Register the plugin with video.js.
videojs.registerComponent("SourceMenuButton", SourceMenuButton);
videojs.registerPlugin("sourceSelector", SourceSelectorPlugin);

/* eslint-disable @typescript-eslint/naming-convention */
declare module "video.js" {
  interface VideoJsPlayer {
    sourceSelector: () => SourceSelectorPlugin;
  }
  interface VideoJsPlayerPluginOptions {
    sourceSelector?: {};
  }
}

export default SourceSelectorPlugin;
