import { LibraryBucketPage } from "../../../components/library/library-bucket-page";

interface CatalogPageProps {
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
}

export default async function CatalogPage({ searchParams }: CatalogPageProps) {
  return (
    <LibraryBucketPage
      basePath="/catalog"
      bucket="catalog"
      searchParams={(await searchParams) ?? {}}
    />
  );
}
