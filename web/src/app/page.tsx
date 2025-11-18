import CreateAuctionForm from "../components/CreateAuctionForm";
import AuctionList from "../components/AuctionList";

export default function HomePage() {
  return (
    <div className="space-y-8">
      <div className="bg-red-950 border border-red-800 text-red-300 rounded-md p-3">
        Note: If the backend was just started, please wait ~50 seconds for it to warm up.
      </div>
      <CreateAuctionForm />
      <AuctionList />
    </div>
  );
}


