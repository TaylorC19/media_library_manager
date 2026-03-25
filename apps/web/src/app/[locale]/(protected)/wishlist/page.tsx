import { LibraryBucketPage } from "../../../../components/library/library-bucket-page";

interface WishlistPageProps {
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
}

export default async function WishlistPage({ searchParams }: WishlistPageProps) {
  return (
    <LibraryBucketPage
      basePath="/wishlist"
      bucket="wishlist"
      searchParams={(await searchParams) ?? {}}
    />
  );
}
