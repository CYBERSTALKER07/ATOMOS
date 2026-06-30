import {
  createCategoryHubMetadata,
  createCategoryHubPage,
} from '@/app/lib/explore/createTopicPage';

const categoryId = 'apps-deploy' as const;

export const generateMetadata = createCategoryHubMetadata(categoryId);
export default createCategoryHubPage(categoryId);
