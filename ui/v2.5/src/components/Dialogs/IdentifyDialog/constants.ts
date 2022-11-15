import * as GQL from "src/core/generated-graphql";

export interface IScraperSource {
  id: string;
  displayName: string;
  stash_box_endpoint?: string;
  scraper_id?: string;
  options?: GQL.IdentifyMetadataOptionsInput;
}

export const sceneFields = [
  "title",
  "date",
  "details",
  "url",
  "code",
  "director",
  "studio",
  "performers",
  "tags",
  "stash_ids",
] as const;
export type SceneField = typeof sceneFields[number];

export const multiValueSceneFields: SceneField[] = [
  "studio",
  "performers",
  "tags",
];

export function sceneFieldMessageID(field: SceneField) {
  if (field === "code") {
    return "scene_code";
  }

  return field;
}
