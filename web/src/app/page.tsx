import CreateAuctionForm from "../components/CreateAuctionForm";
import AuctionList from "../components/AuctionList";
import NoticeBanner from "../components/NoticeBanner";

export default function HomePage() {
  return (
    <div className="space-y-8">
      <NoticeBanner />
      <CreateAuctionForm />
      <AuctionList />
    </div>
  );
}


