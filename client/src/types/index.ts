export type Idea = {
  id: string;
  board_id: string;
  text: string;
  count: number;
  created_by: string;
  created_at: string;
};

export type BrainstormResult = {
  added: number;
  ideas: Idea[];
};

export type WsMessage = {
  type: "idea.added" | "idea.updated";
  data: Idea;
};
