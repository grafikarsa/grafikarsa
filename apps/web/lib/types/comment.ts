import { User } from './user';

export interface Comment {
    id: string;
    content: string;
    created_at: string;
    updated_at: string;
    user: User;
    children: Comment[];
    parent_id?: string; // Optional context
    is_edited?: boolean; // If we add this field
}

export interface CreateCommentRequest {
    portfolio_id: string;
    parent_id?: string;
    content: string;
}
