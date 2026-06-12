declare module "segmentit" {
  interface Token {
    w: string;
    p: number;
  }

  class Segment {
    doSegment(text: string): Token[];
  }

  function useDefault(segment: Segment): void;

  export { Segment, useDefault };
}
