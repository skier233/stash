import React from "react";
import * as GQL from "src/core/generated-graphql";
import { MovieList } from "src/components/Movies/MovieList";
import { useStudioFilterHook } from "src/core/studios";

interface IStudioMoviesPanel {
  studio: GQL.StudioDataFragment;
}

export const StudioMoviesPanel: React.FC<IStudioMoviesPanel> = ({ studio }) => {
  const filterHook = useStudioFilterHook(studio);
  return <MovieList filterHook={filterHook} />;
};
