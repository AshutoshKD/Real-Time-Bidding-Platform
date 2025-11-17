import CreateAuctionForm from "../components/CreateAuctionForm";
import AuctionList from "../components/AuctionList";

export default function HomePage() {
  return (
    <div className="space-y-8">
      <CreateAuctionForm />
      <AuctionList />
    </div>
  );
}


