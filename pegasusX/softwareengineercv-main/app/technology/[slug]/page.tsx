import {
  createTopicMetadata,
  createTopicPage,
  createTopicStaticParams,
} from '@/app/lib/explore/createTopicPage';

const categoryId = 'technology' as const;

export const generateStaticParams = createTopicStaticParams(categoryId);
export const generateMetadata = createTopicMetadata(categoryId);
export default createTopicPage(categoryId);
