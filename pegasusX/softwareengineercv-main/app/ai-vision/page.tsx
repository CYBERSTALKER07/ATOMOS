import {
  createCategoryHubMetadata,
  createCategoryHubPage,
} from '@/app/lib/explore/createTopicPage';

const categoryId = 'ai-vision' as const;

export const generateMetadata = createCategoryHubMetadata(categoryId);
export default createCategoryHubPage(categoryId);
